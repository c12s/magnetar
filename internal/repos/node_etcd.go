package repos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/c12s/magnetar/internal/domain"
	"github.com/juliangruber/go-intersect"
	etcd "go.etcd.io/etcd/client/v3"
	"golang.org/x/exp/slices"
)

// data model
// for get operations
// key - nodes/pool/{nodeId} | nodes/orgs/{orgId}/{nodeId}
// value - protobuf node (id + org + labels)
// for query operations
// key - labels/pool/{labelKey}/{nodeId} | labels/orgs/{orgId}/{labelKey}/{nodeId}
// value - protobuf label (key + value)
// desired OS image (own key: status reports rewrite the node record, a
// read-modify-write there would race with and drop operator intent)
// key - desired_os/{nodeId}
// value - json DesiredOsImage

type nodeEtcdRepo struct {
	etcd            *etcd.Client
	nodeMarshaller  domain.NodeMarshaller
	labelMarshaller domain.LabelMarshaller
}

func NewNodeEtcdRepo(etcd *etcd.Client, nodeMarshaller domain.NodeMarshaller, labelMarshaller domain.LabelMarshaller) (domain.NodeRepo, error) {
	return &nodeEtcdRepo{
		etcd:            etcd,
		nodeMarshaller:  nodeMarshaller,
		labelMarshaller: labelMarshaller,
	}, nil
}

func (n nodeEtcdRepo) Put(node domain.Node) error {
	node.DesiredOsImage = nil // stored under its own key, see PutDesiredOsImage
	err := n.putNodeGetModel(node)
	if err != nil {
		return err
	}
	return n.putNodeQueryModel(node)
}

func (n nodeEtcdRepo) Delete(node domain.Node) error {
	err := n.deleteNodeGetModel(node)
	if err != nil {
		return err
	}
	return n.deleteNodeQueryModel(node)
}

func (n nodeEtcdRepo) Get(nodeId domain.NodeId, org string) (*domain.Node, error) {
	key := getKey(domain.Node{Id: nodeId, Org: org})
	resp, err := n.etcd.Get(context.TODO(), key)
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, domain.ErrNodeNotFound
	}
	node, err := n.nodeMarshaller.Unmarshal(resp.Kvs[0].Value)
	if err != nil {
		return nil, err
	}
	return n.withView(node)
}

func (n nodeEtcdRepo) FindById(nodeId domain.NodeId) (*domain.Node, error) {
	node, err := n.Get(nodeId, "")
	if !errors.Is(err, domain.ErrNodeNotFound) {
		return node, err
	}
	return n.findInOrgs(nodeId)
}

func (n nodeEtcdRepo) findInOrgs(nodeId domain.NodeId) (*domain.Node, error) {
	prefix := fmt.Sprintf("%s/orgs/", getKeyPrefix)
	resp, err := n.etcd.Get(context.TODO(), prefix, etcd.WithPrefix())
	if err != nil {
		return nil, err
	}
	suffix := fmt.Sprintf("/%s", nodeId.Value)
	for _, kv := range resp.Kvs {
		if !strings.HasSuffix(string(kv.Key), suffix) {
			continue
		}
		node, err := n.nodeMarshaller.Unmarshal(kv.Value)
		if err != nil {
			return nil, err
		}
		return n.withView(node)
	}
	return nil, domain.ErrNodeNotFound
}

func (n nodeEtcdRepo) ListNodePool() ([]domain.Node, error) {
	keyPrefix := fmt.Sprintf("%s/pool", getKeyPrefix)
	return n.listNodes(keyPrefix)
}

func (n nodeEtcdRepo) ListOrgOwnedNodes(org string) ([]domain.Node, error) {
	keyPrefix := fmt.Sprintf("%s/orgs/%s", getKeyPrefix, org)
	return n.listNodes(keyPrefix)
}

func (n nodeEtcdRepo) ListAllNodes() ([]domain.Node, error) {
	return n.listNodes(getKeyPrefix)
}

func (n nodeEtcdRepo) QueryNodePool(query domain.Query) ([]domain.Node, error) {
	keyPrefix := fmt.Sprintf("%s/pool", queryKeyPrefix)
	if len(query) == 0 {
		return n.listNodes(keyPrefix)
	}
	nodeIds, err := n.queryNodes(query, keyPrefix)
	if err != nil {
		return nil, err
	}
	nodes := make([]domain.Node, 0)
	for _, nodeId := range nodeIds {
		node, err := n.Get(nodeId, "")
		if err != nil {
			log.Println(err)
			continue
		}
		nodes = append(nodes, *node)
	}
	return nodes, nil
}

func (n nodeEtcdRepo) QueryOrgOwnedNodes(query domain.Query, org string) ([]domain.Node, error) {
	keyPrefix := fmt.Sprintf("%s/orgs/%s", queryKeyPrefix, org)
	if len(query) == 0 {
		return n.ListOrgOwnedNodes(org)
	}
	nodeIds, err := n.queryNodes(query, keyPrefix)
	if err != nil {
		return nil, err
	}
	nodes := make([]domain.Node, 0)
	for _, nodeId := range nodeIds {
		node, err := n.Get(nodeId, org)
		if err != nil {
			log.Println(err)
			continue
		}
		nodes = append(nodes, *node)
	}
	return nodes, nil
}

func (n nodeEtcdRepo) listNodes(keyPrefix string) ([]domain.Node, error) {
	nodes := make([]domain.Node, 0)
	resp, err := n.etcd.Get(context.TODO(), keyPrefix, etcd.WithPrefix())
	if err != nil {
		return nil, err
	}
	desired, err := n.allDesiredOs()
	if err != nil {
		return nil, err
	}
	seen, err := n.ListSeen()
	if err != nil {
		return nil, err
	}
	for _, kv := range resp.Kvs {
		node, err := n.nodeMarshaller.Unmarshal(kv.Value)
		if err != nil {
			return nil, err
		}
		node.DesiredOsImage = desired[node.Id.Value]
		node.LastSeen = seen[node.Id.Value]
		nodes = append(nodes, *node)
	}
	return nodes, nil
}

// PutDesiredOsImage records the OS image the operator declared for a node;
// nil clears it.
func (n nodeEtcdRepo) PutDesiredOsImage(nodeId domain.NodeId, image *domain.DesiredOsImage) error {
	key := desiredOsKey(nodeId)
	if image == nil {
		_, err := n.etcd.Delete(context.TODO(), key)
		return err
	}
	value, err := json.Marshal(image)
	if err != nil {
		return err
	}
	_, err = n.etcd.Put(context.TODO(), key, string(value))
	return err
}

func (n nodeEtcdRepo) GetDesiredOsImage(nodeId domain.NodeId) (*domain.DesiredOsImage, error) {
	resp, err := n.etcd.Get(context.TODO(), desiredOsKey(nodeId))
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, nil
	}
	image := &domain.DesiredOsImage{}
	if err := json.Unmarshal(resp.Kvs[0].Value, image); err != nil {
		return nil, err
	}
	return image, nil
}

func (n nodeEtcdRepo) withDesiredOs(node *domain.Node) (*domain.Node, error) {
	image, err := n.GetDesiredOsImage(node.Id)
	if err != nil {
		return nil, err
	}
	node.DesiredOsImage = image
	return node, nil
}

// withView fills the read-time fields of a node: desired OS intent (its own
// key) and liveness (its own key).
func (n nodeEtcdRepo) withView(node *domain.Node) (*domain.Node, error) {
	if _, err := n.withDesiredOs(node); err != nil {
		return nil, err
	}
	seen, err := n.LastSeen(node.Id)
	if err != nil {
		return nil, err
	}
	node.LastSeen = seen
	return node, nil
}

// LastSeen returns the unix seconds of the node's last heartbeat/update,
// or 0 when the node has never been seen.
func (n nodeEtcdRepo) LastSeen(nodeId domain.NodeId) (int64, error) {
	resp, err := n.etcd.Get(context.TODO(), lastSeenKey(nodeId))
	if err != nil {
		return 0, err
	}
	if resp.Count == 0 {
		return 0, nil
	}
	at, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
	if err != nil {
		return 0, nil // malformed value: treat as never seen
	}
	return at, nil
}

func (n nodeEtcdRepo) TouchSeen(nodeId domain.NodeId, at int64) error {
	_, err := n.etcd.Put(context.TODO(), lastSeenKey(nodeId), strconv.FormatInt(at, 10))
	return err
}

func (n nodeEtcdRepo) DeleteSeen(nodeId domain.NodeId) error {
	_, err := n.etcd.Delete(context.TODO(), lastSeenKey(nodeId))
	return err
}

func (n nodeEtcdRepo) ListSeen() (map[string]int64, error) {
	resp, err := n.etcd.Get(context.TODO(), lastSeenKeyPrefix+"/", etcd.WithPrefix())
	if err != nil {
		return nil, err
	}
	seen := make(map[string]int64, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		at, err := strconv.ParseInt(string(kv.Value), 10, 64)
		if err != nil {
			continue
		}
		seen[strings.TrimPrefix(string(kv.Key), lastSeenKeyPrefix+"/")] = at
	}
	return seen, nil
}

// DeleteNode removes everything that belongs to a node. It is the explicit
// (operator or GC) removal path; Delete above only moves a record between
// the pool and an org during a claim.
func (n nodeEtcdRepo) DeleteNode(node domain.Node) error {
	if err := n.deleteNodeGetModel(node); err != nil {
		return err
	}
	if err := n.deleteNodeQueryModel(node); err != nil {
		return err
	}
	if err := n.DeleteSeen(node.Id); err != nil {
		return err
	}
	return n.PutDesiredOsImage(node.Id, nil)
}

func (n nodeEtcdRepo) allDesiredOs() (map[string]*domain.DesiredOsImage, error) {
	resp, err := n.etcd.Get(context.TODO(), desiredOsKeyPrefix+"/", etcd.WithPrefix())
	if err != nil {
		return nil, err
	}
	images := make(map[string]*domain.DesiredOsImage, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		image := &domain.DesiredOsImage{}
		if err := json.Unmarshal(kv.Value, image); err != nil {
			return nil, err
		}
		images[strings.TrimPrefix(string(kv.Key), desiredOsKeyPrefix+"/")] = image
	}
	return images, nil
}

func desiredOsKey(nodeId domain.NodeId) string {
	return fmt.Sprintf("%s/%s", desiredOsKeyPrefix, nodeId.Value)
}

func (n nodeEtcdRepo) PutLabel(node domain.Node, label domain.Label) (*domain.Node, error) {
	err := n.putLabelGetModel(node, label)
	if err != nil {
		return nil, err
	}
	err = n.putLabelQueryModel(node, label)
	if err != nil {
		return nil, err
	}
	return n.Get(node.Id, node.Org)
}

func (n nodeEtcdRepo) DeleteLabel(node domain.Node, labelKey string) (*domain.Node, error) {
	err := n.deleteLabelGetModel(node, labelKey)
	if err != nil {
		return nil, err
	}
	err = n.deleteLabelQueryModel(node, labelKey)
	if err != nil {
		return nil, err
	}
	return n.Get(node.Id, node.Org)
}

func (n nodeEtcdRepo) deleteNodeGetModel(node domain.Node) error {
	key := getKey(node)
	_, err := n.etcd.Delete(context.TODO(), key)
	return err
}

func (n nodeEtcdRepo) deleteNodeQueryModel(node domain.Node) (err error) {
	for _, label := range node.Labels {
		key := queryKey(node, label.Key())
		_, delErr := n.etcd.Delete(context.TODO(), key)
		err = errors.Join(err, delErr)
	}
	return err
}

func (n nodeEtcdRepo) putNodeGetModel(node domain.Node) error {
	nodeMarshalled, err := n.nodeMarshaller.Marshal(node)
	if err != nil {
		return err
	}
	key := getKey(node)
	_, err = n.etcd.Put(context.TODO(), key, string(nodeMarshalled))
	return err
}

func (n nodeEtcdRepo) putLabelGetModel(node domain.Node, label domain.Label) error {
	labelIndex := -1
	for i, nodeLabel := range node.Labels {
		if nodeLabel.Key() == label.Key() {
			labelIndex = i
		}
	}
	if labelIndex >= 0 {
		node.Labels[labelIndex] = label
	} else {
		node.Labels = append(node.Labels, label)
	}
	return n.putNodeGetModel(node)
}

func (n nodeEtcdRepo) deleteLabelGetModel(node domain.Node, labelKey string) error {
	labelIndex := findIndexByLabelKey(node, labelKey)
	if labelIndex >= 0 {
		node.Labels = slices.Delete(node.Labels, labelIndex, labelIndex+1)
		return n.putNodeGetModel(node)
	}
	return domain.ErrNotFound("label")
}

func (n nodeEtcdRepo) putNodeQueryModel(node domain.Node) error {
	for _, label := range node.Labels {
		err := n.putLabelQueryModel(node, label)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n nodeEtcdRepo) putLabelQueryModel(node domain.Node, label domain.Label) error {
	labelMarshalled, err := n.labelMarshaller.Marshal(label)
	if err != nil {
		return err
	}
	key := queryKey(node, label.Key())
	_, err = n.etcd.Put(context.TODO(), key, string(labelMarshalled))
	return err
}

func (n nodeEtcdRepo) deleteLabelQueryModel(node domain.Node, labelKey string) error {
	key := queryKey(node, labelKey)
	_, err := n.etcd.Delete(context.TODO(), key)
	return err
}

func (n nodeEtcdRepo) queryNodes(query domain.Query, keyPrefix string) ([]domain.NodeId, error) {
	nodeIds := make([]domain.NodeId, 0)
	for i, selector := range query {
		currNodes, err := n.selectNodes(selector, keyPrefix)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			nodeIds = currNodes
		} else {
			intersection := intersect.Simple(nodeIds, currNodes)
			nodeIds = make([]domain.NodeId, len(intersection))
			for i, node := range intersection {
				nodeIds[i] = node.(domain.NodeId)
			}
		}
	}
	return nodeIds, nil
}

func (n nodeEtcdRepo) selectNodes(selector domain.Selector, keyPrefix string) ([]domain.NodeId, error) {
	prefix := fmt.Sprintf("%s/%s/", keyPrefix, selector.LabelKey)
	resp, err := n.etcd.Get(context.TODO(), prefix, etcd.WithPrefix())
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, domain.ErrNotFound("nodes")
	}

	nodeIds := make([]domain.NodeId, 0)
	for _, kv := range resp.Kvs {
		nodeLabel, err := n.labelMarshaller.Unmarshal(kv.Value)
		if err != nil {
			return nil, err
		}
		cmpResult, err := nodeLabel.Compare(selector.Value)
		if err != nil {
			log.Println(err)
			continue
		}
		if slices.Contains(cmpResult, selector.ShouldBe) {
			nodeId := extractNodeIdFromQueryKey(string(kv.Key))
			nodeIds = append(nodeIds, domain.NodeId{
				Value: nodeId,
			})
		}
	}
	return nodeIds, nil
}

const (
	getKeyPrefix       = "nodes"
	queryKeyPrefix     = "labels"
	desiredOsKeyPrefix = "desired_os"
	// lastSeenKeyPrefix is deliberately outside the nodes/ prefix: listNodes
	// scans nodes/ and must never try to unmarshal liveness values as nodes.
	lastSeenKeyPrefix = "last_seen"
)

func lastSeenKey(nodeId domain.NodeId) string {
	return fmt.Sprintf("%s/%s", lastSeenKeyPrefix, nodeId.Value)
}

func getKey(node domain.Node) string {
	if node.Claimed() {
		return fmt.Sprintf("%s/orgs/%s/%s", getKeyPrefix, node.Org, node.Id.Value)
	}
	return fmt.Sprintf("%s/pool/%s", getKeyPrefix, node.Id.Value)
}

func queryKey(node domain.Node, labelKey string) string {
	if node.Claimed() {
		return fmt.Sprintf("%s/orgs/%s/%s/%s", queryKeyPrefix, node.Org, labelKey, node.Id.Value)
	}
	return fmt.Sprintf("%s/pool/%s/%s", queryKeyPrefix, labelKey, node.Id.Value)
}

func extractNodeIdFromQueryKey(key string) string {
	if strings.HasPrefix(key, fmt.Sprintf("%s/pool", queryKeyPrefix)) {
		return strings.Split(key, "/")[3]
	}
	return strings.Split(key, "/")[4]
}

func findIndexByLabelKey(node domain.Node, labelKey string) int {
	labelIndex := -1
	for i, nodeLabel := range node.Labels {
		if nodeLabel.Key() == labelKey {
			labelIndex = i
			break
		}
	}

	return labelIndex

}
