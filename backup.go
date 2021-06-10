package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/backup"
	"github.com/vmware-tanzu/velero/pkg/builder"
)

func (cxn clusterConnection) backup(ctx context.Context, log *logrus.Logger, config backupConfig) error {
	backupResource := builder.
		ForBackup(config.namespace, config.name).
		IncludedNamespaces(config.includedNamespaces).
		DefaultVolumesToRestic(false).
		Result()

	request := backup.Request{
		Backup: backupResource,
	}

	os.MkdirAll(path.Dir(config.filepath), 0644)
	backupFile, err := os.Create(config.filepath)
	if err != nil {
		return fmt.Errorf("create backup file: %w", err)
	}
	defer backupFile.Close()

	backupper, err := backup.NewKubernetesBackupper(
		cxn.veleroClient.VeleroV1(),
		cxn.discoveryHelper,
		cxn.dynamicFactory,
		cxn.podCommandExecutor,
		nil,
		0,
		false,
	)
	if err != nil {
		return fmt.Errorf("create backupper: %w", err)
	}

	backupper.Backup(log, &request, backupFile, nil, nil)

	return nil
}
