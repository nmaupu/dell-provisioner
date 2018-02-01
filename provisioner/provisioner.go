package provisioner

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"strconv"
	"strings"
)

const (
	DefaultFS = "ext4"
)

var (
	// dellProvisioner is an implem of controller.Provisioner
	_ controller.Provisioner = &dellProvisioner{}
)

type dellProvisioner struct {
	Identifier   string
	SanAddress   string
	SanGroupName string
	SanPassword  string
	SmcliCommand string
}

func New(identifier, sanAddress, sanGroupName, sanPassword, smcliCommand string) controller.Provisioner {
	return &dellProvisioner{
		Identifier:   identifier,
		SanAddress:   sanAddress,
		SanGroupName: sanGroupName,
		SanPassword:  sanPassword,
		SmcliCommand: smcliCommand,
	}
}

// getAccessModes returns access modes iscsi volumes support
func (p *dellProvisioner) getAccessModes() []v1.PersistentVolumeAccessMode {
	return []v1.PersistentVolumeAccessMode{
		v1.ReadWriteOnce,
		v1.ReadOnlyMany,
	}
}

func (p *dellProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	if !accessModesContainedInAll(p.getAccessModes(), options.PVC.Spec.AccessModes) {
		return nil, fmt.Errorf("invalid AccessModes %v: only AccessModes %v are supported", options.PVC.Spec.AccessModes, p.getAccessModes())
	}

	glog.Infoln("New provision request received for pvc: ", options.PVName)

	var portals []string
	if len(options.Parameters["portals"]) > 0 {
		portals = strings.Split(options.Parameters["portals"], ",")
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: options.PVName,
			Annotations: map[string]string{
				"dellProvisionerIdentifier": p.Identifier,
			},
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceName(v1.ResourceStorage): options.PVC.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)],
			},
			PersistentVolumeSource: v1.PersistentVolumeSource{
				ISCSI: &v1.ISCSIVolumeSource{
					TargetPortal:   options.Parameters["targetPortal"],
					Portals:        portals,
					IQN:            options.Parameters["iqn"],
					ISCSIInterface: options.Parameters["iscsiInterface"],
					Lun:            0,
					ReadOnly:       getReadOnly(options.Parameters["readonly"]),
					FSType:         getFsType(options.Parameters["fsType"]),
				},
			},
		},
	}

	return pv, nil
}

func (p *dellProvisioner) Delete(volume *v1.PersistentVolume) error {
	fmt.Printf("Deleting %s, %s, %s\n", volume.Name, p.SanAddress, p.SanGroupName)
	return nil
}

func getReadOnly(readonly string) bool {
	isReadOnly, err := strconv.ParseBool(readonly)
	if err != nil {
		return false
	}
	return isReadOnly
}

func getFsType(fsType string) string {
	if fsType == "" {
		return DefaultFS
	}
	return fsType
}

// AccessModesContains returns whether the requested mode is contained by modes
func accessModesContains(modes []v1.PersistentVolumeAccessMode, mode v1.PersistentVolumeAccessMode) bool {
	for _, m := range modes {
		if m == mode {
			return true
		}
	}
	return false
}

func accessModesContainedInAll(indexedModes []v1.PersistentVolumeAccessMode, requestedModes []v1.PersistentVolumeAccessMode) bool {
	for _, mode := range requestedModes {
		if !accessModesContains(indexedModes, mode) {
			return false
		}
	}
	return true
}
