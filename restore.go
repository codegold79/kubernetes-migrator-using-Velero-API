package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/builder"
	"github.com/vmware-tanzu/velero/pkg/restore"
)

var defaultRestorePriorities = []string{
	"customresourcedefinitions",
	"namespaces",
	"storageclasses",
	"volumesnapshotclass.snapshot.storage.k8s.io",
	"volumesnapshotcontents.snapshot.storage.k8s.io",
	"volumesnapshots.snapshot.storage.k8s.io",
	"persistentvolumes",
	"persistentvolumeclaims",
	"secrets",
	"configmaps",
	"serviceaccounts",
	"limitranges",
	"pods",
	// we fully qualify replicasets.apps because prior to Kubernetes 1.16,
	// replicasets also existed in the extensions API group, but we back up
	// replicasets from "apps" so we want to ensure that we prioritize restoring
	// from "apps" too, since this is how they're stored in the backup.
	"replicasets.apps",
	"clusters.cluster.x-k8s.io",
	"clusterresourcesets.addons.cluster.x-k8s.io",
}

func (cxn clusterConnection) restore(ctx context.Context, log *logrus.Logger, config restoreConfig) error {
	log.WithField("event", "restore")

	backupResource := builder.
		ForBackup(namespace, config.name).
		IncludedNamespaces(config.includedNamespaces).
		DefaultVolumesToRestic(false).
		Result()

	restoreResource := builder.
		ForRestore(config.namespace, config.name).
		Backup(config.backupName).
		Result()

	log.WithField("file", config.filepath).Info("open backup file")
	backupFile, err := os.Open(config.filepath)
	if err != nil {
		return fmt.Errorf("open backup file: %w", err)
	}

	request := restore.Request{
		Log:          log,
		Backup:       backupResource,
		Restore:      restoreResource,
		BackupReader: backupFile,
	}

	restorer, err := restore.NewKubernetesRestorer(
		cxn.veleroClient.VeleroV1(),
		cxn.discoveryHelper,
		cxn.dynamicFactory,
		defaultRestorePriorities,
		cxn.kubeClient.CoreV1().Namespaces(),
		nil,
		0,
		config.resourceTerminatingTimeout,
		log,
		cxn.podCommandExecutor,
		cxn.kubeClient.CoreV1().RESTClient(),
	)
	if err != nil {
		return fmt.Errorf("create restorer: %w", err)
	}

	restorer.Restore(request, nil, nil, nil)

	return nil
}
