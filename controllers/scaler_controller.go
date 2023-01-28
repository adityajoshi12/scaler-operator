/*
Copyright 2022 Aditya Joshi.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	apiv1alpha1 "github.com/adityajoshi12/scaler-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("controller_scaler")

// ScalerReconciler reconciles a Scaler object
type ScalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=api.adityajoshi.online,resources=scalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.adityajoshi.online,resources=scalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.adityajoshi.online,resources=scalers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Scaler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ScalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := logger.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)

	log.Info("Reconcile called")
	scaler := &apiv1alpha1.Scaler{}

	err := r.Get(ctx, req.NamespacedName, scaler)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Scaler resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed")
		return ctrl.Result{}, err
	}

	startTime := scaler.Spec.Start
	endTime := scaler.Spec.End

	// current time in UTC
	currentHour := time.Now().UTC().Hour()
	log.Info(fmt.Sprintf("current time in hour : %d\n", currentHour))

	if currentHour >= startTime && currentHour <= endTime {

		if err = scaleDeployment(scaler, r, ctx, int32(scaler.Spec.Replicas)); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Duration(30 * time.Second)}, nil
}

func scaleDeployment(scaler *apiv1alpha1.Scaler, r *ScalerReconciler, ctx context.Context, replicas int32) error {
	for _, deploy := range scaler.Spec.Deployments {
		dep := &v1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: deploy.Namespace,
			Name:      deploy.Name,
		}, dep)
		if err != nil {
			return err
		}

		if dep.Spec.Replicas != &replicas {
			dep.Spec.Replicas = &replicas
			err := r.Update(ctx, dep)
			if err != nil {
				scaler.Status.Status = apiv1alpha1.FAILED
				return err
			}
			scaler.Status.Status = apiv1alpha1.SUCCESS
			err = r.Status().Update(ctx, scaler)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Scaler{}).
		Complete(r)
}
