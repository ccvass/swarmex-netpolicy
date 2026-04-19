package netpolicy

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

const (
	labelAllow     = "swarmex.netpolicy.allow"     // comma-separated service names to allow cross-namespace communication
	labelNamespace = "swarmex.namespace"            // read from namespaces controller
	networkPrefix  = "ns-"
)

// Controller manages cross-namespace network policies.
// Works WITH swarmex-namespaces: namespaces isolate (separate overlay networks),
// netpolicy allows specific cross-namespace connections by adding namespace networks.
type Controller struct {
	client  *client.Client
	logger  *slog.Logger
	pending map[string]bool
	mu      sync.Mutex
}

func New(cli *client.Client, logger *slog.Logger) *Controller {
	return &Controller{client: cli, logger: logger, pending: make(map[string]bool)}
}

func (c *Controller) HandleEvent(ctx context.Context, event events.Message) {
	if event.Type != events.ServiceEventType {
		return
	}
	if event.Action != events.ActionCreate && event.Action != events.ActionUpdate {
		return
	}
	c.mu.Lock()
	if c.pending[event.Actor.ID] {
		c.mu.Unlock()
		return
	}
	c.pending[event.Actor.ID] = true
	c.mu.Unlock()

	go func() {
		time.Sleep(3 * time.Second) // debounce — wait for namespaces controller to assign network first
		c.reconcile(ctx, event.Actor.ID)
		c.mu.Lock()
		delete(c.pending, event.Actor.ID)
		c.mu.Unlock()
	}()
}

func (c *Controller) reconcile(ctx context.Context, serviceID string) {
	svc, _, err := c.client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return
	}

	allowList := svc.Spec.Labels[labelAllow]
	if allowList == "" {
		return
	}

	allServices, err := c.client.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return
	}

	networksToAdd := make(map[string]string)
	for _, name := range strings.Split(allowList, ",") {
		name = strings.TrimSpace(name)
		ns := c.findServiceNamespace(name, allServices)
		if ns == "" {
			continue
		}
		netName := networkPrefix + ns
		netID := c.resolveNetworkID(ctx, netName)
		if netID != "" {
			networksToAdd[netID] = netName
		}
	}

	if len(networksToAdd) == 0 {
		return
	}

	// Re-inspect to get latest version (avoids "update out of sequence")
	svc, _, err = c.client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return
	}

	existing := make(map[string]bool)
	for _, n := range svc.Spec.TaskTemplate.Networks {
		existing[n.Target] = true
	}

	added := []string{}
	for netID, netName := range networksToAdd {
		if !existing[netID] {
			svc.Spec.TaskTemplate.Networks = append(svc.Spec.TaskTemplate.Networks,
				swarm.NetworkAttachmentConfig{Target: netID})
			added = append(added, netName)
		}
	}

	if len(added) == 0 {
		return // already has all required networks
	}

	_, err = c.client.ServiceUpdate(ctx, serviceID, svc.Version, svc.Spec, types.ServiceUpdateOptions{})
	if err != nil {
		c.logger.Error("netpolicy update failed", "service", svc.Spec.Name, "error", err)
		return
	}
	c.logger.Info("netpolicy applied — cross-namespace access granted",
		"service", svc.Spec.Name, "added_networks", added)
}

func (c *Controller) findServiceNamespace(name string, services []swarm.Service) string {
	for _, svc := range services {
		if svc.Spec.Name == name {
			return svc.Spec.Labels[labelNamespace]
		}
	}
	return ""
}

func (c *Controller) resolveNetworkID(ctx context.Context, name string) string {
	net, err := c.client.NetworkInspect(ctx, name, types.NetworkInspectOptions{})
	if err != nil {
		return ""
	}
	return net.ID
}
