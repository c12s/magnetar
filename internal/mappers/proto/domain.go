package proto

import (
	"errors"

	"github.com/c12s/magnetar/internal/domain"
	"github.com/c12s/magnetar/pkg/api"
	"github.com/golang/protobuf/proto"
)

func LabelFromDomain(label domain.Label) (*api.Label, error) {
	value, err := ValueFromDomain(label.Value())
	if err != nil {
		return nil, err
	}
	return &api.Label{
		Key:   label.Key(),
		Value: value,
	}, nil
}

func LabelToDomain(l *api.Label) (domain.Label, error) {
	var label domain.Label
	var err error
	switch l.Value.Type {
	case api.Value_Bool:
		protoValue := &api.BoolValue{}
		err = proto.Unmarshal(l.Value.Marshalled, protoValue)
		if err == nil {
			label = domain.NewBoolLabel(l.Key, protoValue.Value)
		}
	case api.Value_Float64:
		protoValue := &api.Float64Value{}
		err = proto.Unmarshal(l.Value.Marshalled, protoValue)
		if err == nil {
			label = domain.NewFloat64Label(l.Key, protoValue.Value)
		}
	case api.Value_String:
		protoValue := &api.StringValue{}
		err = proto.Unmarshal(l.Value.Marshalled, protoValue)
		if err == nil {
			label = domain.NewStringLabel(l.Key, protoValue.Value)
		}
	default:
		err = errors.New("unsupported data type")
	}
	return label, err
}

func ValueFromDomain(value interface{}) (*api.Value, error) {
	var marshalled []byte
	var valueType api.Value_ValueTYpe
	var err error
	switch value := value.(type) {
	case bool:
		marshalled, err = proto.Marshal(&api.BoolValue{Value: value})
		valueType = api.Value_Bool
	case float64:
		marshalled, err = proto.Marshal(&api.Float64Value{Value: value})
		valueType = api.Value_Float64
	case string:
		marshalled, err = proto.Marshal(&api.StringValue{Value: value})
		valueType = api.Value_String
	default:
		err = errors.New("unsupported data type")
	}
	return &api.Value{
		Marshalled: marshalled,
		Type:       valueType,
	}, err
}

func NodeStringifiedFromDomain(node domain.Node) (*api.NodeStringified, error) {
	labels := make([]*api.LabelStringified, len(node.Labels))
	for i, label := range node.Labels {
		labelProto := &api.LabelStringified{}
		labelProto, err := LabelStringifiedFromDomain(label)
		if err != nil {
			return nil, err
		}
		labels[i] = labelProto
	}
	osStatus := OsStatusFromDomain(node.OsStatus)
	return &api.NodeStringified{
		Id:             node.Id.Value,
		Org:            node.Org,
		Labels:         labels,
		Resources:      node.Resources,
		Capabilities:   node.Capabilities,
		OsStatus:       osStatus,
		DesiredOsImage: DesiredOsImageFromDomain(node.DesiredOsImage),
		LastSeen:       node.LastSeen,
		Liveness:       node.Liveness,
	}, nil
}

func LabelStringifiedFromDomain(label domain.Label) (*api.LabelStringified, error) {
	return &api.LabelStringified{
		Key:   label.Key(),
		Value: label.StringValue(),
	}, nil
}

func NodeFromDomain(node domain.Node) (*api.Node, error) {
	resp := &api.Node{
		Id:             node.Id.Value,
		Org:            node.Org,
		Labels:         make([]*api.Label, len(node.Labels)),
		Resources:      node.Resources,
		BindAddress:    node.BindAddress,
		Capabilities:   node.Capabilities,
		OsStatus:       OsStatusFromDomain(node.OsStatus),
		DesiredOsImage: DesiredOsImageFromDomain(node.DesiredOsImage),
		DesiredVms:     make([]*api.DesiredVm, len(node.DesiredVms)),
		VmStatus:       make([]*api.VmStatus, len(node.VmStatus)),
	}
	for i, label := range node.Labels {
		protoLabel, err := LabelFromDomain(label)
		if err != nil {
			return nil, err
		}
		resp.Labels[i] = protoLabel
	}
	for i, vm := range node.DesiredVms {
		resp.DesiredVms[i] = DesiredVmFromDomain(vm)
	}
	for i, vm := range node.VmStatus {
		resp.VmStatus[i] = VmStatusFromDomain(vm)
	}
	return resp, nil
}

func OsStatusFromDomain(status *domain.OsStatus) *api.OsStatus {
	if status == nil {
		return nil
	}
	resp := &api.OsStatus{
		BootedImage:    status.BootedImage,
		BootedDigest:   status.BootedDigest,
		BootedVersion:  status.BootedVersion,
		Phase:          status.Phase,
		RollbackImage:  status.RollbackImage,
		RollbackDigest: status.RollbackDigest,
	}
	if status.LastAttempt != nil {
		resp.LastAttempt = &api.OsAttempt{
			Image:     status.LastAttempt.Image,
			Digest:    status.LastAttempt.Digest,
			Result:    status.LastAttempt.Result,
			Timestamp: status.LastAttempt.Timestamp,
		}
	}
	return resp
}

func DesiredOsImageFromDomain(image *domain.DesiredOsImage) *api.DesiredOsImage {
	if image == nil {
		return nil
	}
	return &api.DesiredOsImage{
		ImageRef:     image.ImageRef,
		Digest:       image.Digest,
		RebootPolicy: image.RebootPolicy,
	}
}

func DesiredVmFromDomain(vm domain.DesiredVm) *api.DesiredVm {
	labels := make([]*api.Label, len(vm.Labels))
	for i, label := range vm.Labels {
		protoLabel, err := LabelFromDomain(label)
		if err != nil {
			return nil
		}
		labels[i] = protoLabel
	}
	return &api.DesiredVm{
		Name:     vm.Name,
		Qcow2Url: vm.Qcow2Url,
		Digest:   vm.Digest,
		Vcpus:    vm.Vcpus,
		MemoryMb: vm.MemoryMb,
		Labels:   labels,
	}
}

func VmStatusFromDomain(vm domain.VmStatus) *api.VmStatus {
	return &api.VmStatus{
		Name:     vm.Name,
		State:    vm.State,
		BootedOs: vm.BootedOs,
	}
}

func NodeToDomain(node *api.Node) (*domain.Node, error) {
	resp := &domain.Node{
		Id: domain.NodeId{
			Value: node.Id,
		},
		Org:            node.Org,
		Labels:         make([]domain.Label, len(node.Labels)),
		Resources:      node.Resources,
		BindAddress:    node.BindAddress,
		Capabilities:   node.Capabilities,
		OsStatus:       OsStatusToDomain(node.OsStatus),
		DesiredOsImage: DesiredOsImageToDomain(node.DesiredOsImage),
		DesiredVms:     make([]domain.DesiredVm, len(node.DesiredVms)),
		VmStatus:       make([]domain.VmStatus, len(node.VmStatus)),
	}
	for i, protoLabel := range node.Labels {
		label, err := LabelToDomain(protoLabel)
		if err != nil {
			return nil, err
		}
		resp.Labels[i] = label
	}
	for i, protoVm := range node.DesiredVms {
		vm, err := DesiredVmToDomain(protoVm)
		if err != nil {
			return nil, err
		}
		resp.DesiredVms[i] = *vm
	}
	for i, protoVm := range node.VmStatus {
		resp.VmStatus[i] = *VmStatusToDomain(protoVm)
	}
	return resp, nil
}

func OsStatusToDomain(status *api.OsStatus) *domain.OsStatus {
	if status == nil {
		return nil
	}
	resp := &domain.OsStatus{
		BootedImage:    status.BootedImage,
		BootedDigest:   status.BootedDigest,
		BootedVersion:  status.BootedVersion,
		Phase:          status.Phase,
		RollbackImage:  status.RollbackImage,
		RollbackDigest: status.RollbackDigest,
	}
	if status.LastAttempt != nil {
		resp.LastAttempt = &domain.OsAttempt{
			Image:     status.LastAttempt.Image,
			Digest:    status.LastAttempt.Digest,
			Result:    status.LastAttempt.Result,
			Timestamp: status.LastAttempt.Timestamp,
		}
	}
	return resp
}

func DesiredOsImageToDomain(image *api.DesiredOsImage) *domain.DesiredOsImage {
	if image == nil {
		return nil
	}
	return &domain.DesiredOsImage{
		ImageRef:     image.ImageRef,
		Digest:       image.Digest,
		RebootPolicy: image.RebootPolicy,
	}
}

func DesiredVmToDomain(vm *api.DesiredVm) (*domain.DesiredVm, error) {
	labels := make([]domain.Label, len(vm.Labels))
	for i, protoLabel := range vm.Labels {
		label, err := LabelToDomain(protoLabel)
		if err != nil {
			return nil, err
		}
		labels[i] = label
	}
	return &domain.DesiredVm{
		Name:     vm.Name,
		Qcow2Url: vm.Qcow2Url,
		Digest:   vm.Digest,
		Vcpus:    vm.Vcpus,
		MemoryMb: vm.MemoryMb,
		Labels:   labels,
	}, nil
}

func VmStatusToDomain(vm *api.VmStatus) *domain.VmStatus {
	return &domain.VmStatus{
		Name:     vm.Name,
		State:    vm.State,
		BootedOs: vm.BootedOs,
	}
}

func VmDesiredGetReqToDomain(req *api.VmDesiredGetReq) domain.NodeId {
	return domain.NodeId{Value: req.HostId}
}

func VmDesiredSetReqToDomain(req *api.VmDesiredSetReq) (domain.NodeId, []domain.DesiredVm, error) {
	vms := make([]domain.DesiredVm, len(req.Vms))
	for i, protoVm := range req.Vms {
		vm, err := DesiredVmToDomain(protoVm)
		if err != nil {
			return domain.NodeId{}, nil, err
		}
		vms[i] = *vm
	}
	return domain.NodeId{Value: req.HostId}, vms, nil
}

func VmStatusGetReqToDomain(req *api.VmStatusGetReq) domain.NodeId {
	return domain.NodeId{Value: req.HostId}
}

func VmStatusSetReqToDomain(req *api.VmStatusSetReq) (domain.NodeId, []domain.VmStatus) {
	statuses := make([]domain.VmStatus, len(req.Statuses))
	for i, protoStatus := range req.Statuses {
		statuses[i] = *VmStatusToDomain(protoStatus)
	}
	return domain.NodeId{Value: req.HostId}, statuses
}

func VmDesiredGetRespFromDomain(vms []domain.DesiredVm) *api.VmDesiredGetResp {
	protoVms := make([]*api.DesiredVm, len(vms))
	for i, vm := range vms {
		protoVms[i] = DesiredVmFromDomain(vm)
	}
	return &api.VmDesiredGetResp{Vms: protoVms}
}

func VmDesiredSetRespFromDomain(vms []domain.DesiredVm) *api.VmDesiredSetResp {
	protoVms := make([]*api.DesiredVm, len(vms))
	for i, vm := range vms {
		protoVms[i] = DesiredVmFromDomain(vm)
	}
	return &api.VmDesiredSetResp{Vms: protoVms}
}

func VmStatusGetRespFromDomain(statuses []domain.VmStatus) *api.VmStatusGetResp {
	protoStatuses := make([]*api.VmStatus, len(statuses))
	for i, status := range statuses {
		protoStatuses[i] = VmStatusFromDomain(status)
	}
	return &api.VmStatusGetResp{Statuses: protoStatuses}
}
