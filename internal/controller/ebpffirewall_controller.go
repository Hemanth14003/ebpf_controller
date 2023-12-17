/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/cilium/ebpf"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
	"os/exec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hemanthv1alpha1 "hemanth.vit/ebpfcontroller/api/v1alpha1"
)

const (
	blockMapName = "blocked_ips"
)

// convert ip address into network byte order
func ipToInt(val string) uint32 {
	ip := net.ParseIP(val).To4()
	return binary.LittleEndian.Uint32(ip)
}

func sbnToInt(val string) uint32 {
	ip := net.ParseIP(val).To4()
	return binary.BigEndian.Uint32(ip)
}

// EbpffirewallReconciler reconciles a Ebpffirewall object
type EbpffirewallReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=hemanth.hemanth.vit,resources=ebpffirewalls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=hemanth.hemanth.vit,resources=ebpffirewalls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=hemanth.hemanth.vit,resources=ebpffirewalls/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ebpffirewall object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
var ebpfLoad bool = false

func (r *EbpffirewallReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	if !ebpfLoad {
		cmd := exec.Command("tc", "qdisc", "add", "dev", "ens33", "clsact")
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to add qdisc: %+v", err.Error())
		}
		cmd.Wait()

		cmd = exec.Command("tc", "filter", "add", "dev", "ens33", "ingress", "bpf", "da", "obj", "/home/hemanth1/projects/ebpfcontroller/ebpf.o", "sec", "tc/ingress")
		if err := cmd.Run(); err != nil {
			fmt.Printf("Failed to add tc filter: %+v", err)
		}
		cmd.Wait()

		ebpfLoad = true

	}
	// Loading ebpf map
	loadPinOptions := ebpf.LoadPinOptions{}
	blockMap, err := ebpf.LoadPinnedMap(fmt.Sprintf("/sys/fs/bpf/tc/globals/%s", blockMapName), &loadPinOptions)
	if err != nil {
		fmt.Println("Error:", err)
	}
	existingBlockedIp := make(map[uint32]struct{})
	iter := blockMap.Iterate()
	var key uint32
	var value uint32
	for iter.Next(&key, &value) {
		existingBlockedIp[key] = struct{}{}
	}
	myKind := hemanthv1alpha1.Ebpffirewall{}

	if err := r.Get(ctx, req.NamespacedName, &myKind); err != nil {
		fmt.Println("Error:", err)
		// Ignore NotFound errors as they will be retried automatically if the
		// resource is created in the future.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	for _, ser := range myKind.Spec.Block {
		if net.ParseIP(ser.Ip) != nil {
			ip := ipToInt(ser.Ip)
			if _, exist := existingBlockedIp[ip]; !exist {
				if err := blockMap.Put(&ip, &ip); err != nil { // Adding ip to the map
					fmt.Println("Failed to update the element in the 'blocked_ips' map:", err)
				}
			} else {
				delete(existingBlockedIp, ip)
			}
		} else if net.ParseIP(ser.Subnet) != nil {
			sbn := sbnToInt(ser.Subnet)
			if _, exist := existingBlockedIp[sbn]; !exist {
				if err := blockMap.Put(&sbn, &sbn); err != nil { // Adding ip to the map
					fmt.Println("Failed to update the element in the 'blocked_ips' map:", err)
				}
			} else {
				delete(existingBlockedIp, sbn)
			}
		} else {
			ips, err := net.LookupIP(ser.Dns)
			if err != nil {
				fmt.Printf("DNS lookup failed: %v\n", err)
			}
			for _, ipD := range ips {
				if ipD.To4() != nil {
					//convert ip address into network byte order
					ipNB := ipToInt(ipD.String())
					if _, exist := existingBlockedIp[ipNB]; !exist {
						if err := blockMap.Put(&ipNB, &ipNB); err != nil { // Adding ip to the map
							fmt.Println("Failed to update the element in the 'blocked_ips' map:", err)
						}
					} else {
						delete(existingBlockedIp, ipNB)
					}
				}
			}
		}
	}

	for key, _ := range existingBlockedIp {
		if err := blockMap.Delete(&key); err != nil { // Deleting ip from the map
			fmt.Println("Failed to delete the element from the 'blocked_ips' map:", err)
		}
		delete(existingBlockedIp, key)
	}
	myFinalizerName := "hemanth.hemanth.vit/finalizer"

	if myKind.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&myKind, myFinalizerName) {
			controllerutil.AddFinalizer(&myKind, myFinalizerName)
			if err := r.Update(ctx, &myKind); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&myKind, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			r.deleteExternalResources(&myKind)
			for iter.Next(&key, &value) {
				if err := blockMap.Delete(&key); err != nil { // Deleting ip from the map
					fmt.Println("Failed to delete the element from the 'blocked_ips' map:", err)
				}
			}
			ebpfLoad = false

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&myKind, myFinalizerName)
			if err := r.Update(ctx, &myKind); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}
func (r *EbpffirewallReconciler) deleteExternalResources(myKind *hemanthv1alpha1.Ebpffirewall) {
	//
	// delete any external resources associated with the cronJob
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object.
	cmd := exec.Command("tc", "qdisc", "del", "dev", "ens33", "clsact")
	if err := cmd.Run(); err != nil {
		fmt.Println("Failed to delete qdisc:", err)
	}
	cmd.Wait()

}

// SetupWithManager sets up the controller with the Manager.
func (r *EbpffirewallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hemanthv1alpha1.Ebpffirewall{}).
		Complete(r)
}
