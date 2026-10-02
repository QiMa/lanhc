// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build !plan9

package peerrelay

import (
	"context"
	"fmt"
	"net/netip"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"lanhc.com/ipn"
	tsapi "lanhc.com/k8s-operator/apis/v1alpha1"
	"lanhc.com/k8s-operator/reconciler"
	"lanhc.com/k8s-operator/reconciler/lanhcd"
	"lanhc.com/kube/kubetypes"
)

func configSecretName(prName string, idx int32) string {
	return replicaName(prName, idx) + "-config"
}

func peerRelayHostname(pr *tsapi.PeerRelay, idx int32) string {
	prefix := string(pr.Spec.HostnamePrefix)
	if prefix == "" {
		prefix = pr.Name
	}
	return fmt.Sprintf("%s-%d", prefix, idx)
}

func peerRelayLanhcdConfig(pr *tsapi.PeerRelay, idx int32, endpoints []tsapi.PeerRelayEndpoint, authKey *string) ipn.ConfigVAlpha {
	conf := ipn.ConfigVAlpha{
		Version:         "alpha0",
		AcceptDNS:       "false",
		AcceptRoutes:    "false",
		Locked:          "false",
		Hostname:        new(peerRelayHostname(pr, idx)),
		RelayServerPort: new(uint16(servicePort)),
		AuthKey:         authKey,
	}

	// Advertise every address the replica's load balancer answers on. A load balancer spanning several
	// availability zones has one per zone, and a peer that cannot reach one can still reach the relay through
	// another.
	for _, endpoint := range endpoints {
		addr, err := netip.ParseAddr(endpoint.Address)
		if err != nil {
			continue
		}

		conf.RelayServerStaticEndpoints = append(conf.RelayServerStaticEndpoints, netip.AddrPortFrom(addr, uint16(endpoint.Port)))
	}

	return conf
}

func (r *Reconciler) peerRelayConfigSecret(pr *tsapi.PeerRelay, idx int32, endpoints []tsapi.PeerRelayEndpoint, authKey *string) (*corev1.Secret, error) {
	labels := peerRelayServiceLabels(pr.Name, idx)
	return lanhcd.NewConfigSecret(lanhcd.ConfigSecretOptions{
		Name:      configSecretName(pr.Name, idx),
		Namespace: r.lanhcNamespace,
		Labels:    labels,
		Config:    peerRelayLanhcdConfig(pr, idx, endpoints, authKey),
	})
}

func (r *Reconciler) peerRelayStatefulSet(pr *tsapi.PeerRelay, replicas int32, pc *tsapi.ProxyClass) *appsv1.StatefulSet {
	labels := peerRelayLabels(pr.Name)
	ss := lanhcd.NewStatefulSet(lanhcd.StatefulSetOptions{
		Name:               resourceName(pr.Name),
		Namespace:          r.lanhcNamespace,
		Labels:             labels,
		Image:              r.proxyImage,
		Replicas:           replicas,
		ServiceAccountName: "proxies",
		ConfigSecretNameFunc: func(idx int32) string {
			return configSecretName(pr.Name, idx)
		},
	})

	return lanhcd.ApplyProxyClass(ss, pc, managedLabelKeys, nil)
}

var managedLabelKeys = []string{
	kubetypes.LabelManaged,
	reconciler.LabelParentType,
	reconciler.LabelParentName,
}

func (r *Reconciler) getProxyClass(ctx context.Context, pr *tsapi.PeerRelay) (*tsapi.ProxyClass, error) {
	if pr.Spec.ProxyClass == "" {
		return nil, nil
	}

	var pc tsapi.ProxyClass
	if err := r.Get(ctx, types.NamespacedName{Name: pr.Spec.ProxyClass}, &pc); err != nil {
		return nil, fmt.Errorf("failed to get ProxyClass %q: %w", pr.Spec.ProxyClass, err)
	}
	return &pc, nil
}
