/*
Use Velero API back up and restore Kubernetes clusters remotely.
*/
package main

import (
	"context"
	"errors"
	"flag"
	"path"
	"time"

	"github.com/sirupsen/logrus"
)

// TODO: Have these passed in as flags
const (
	namespace                  = "migrator"
	backupName                 = "backup"
	restoreName                = "restore"
	resourceTerminatingTimeout = time.Minute * 10
)

type backupConfig struct {
	name               string
	namespace          string
	includedNamespaces string
	filepath           string
}

type restoreConfig struct {
	name                       string
	namespace                  string
	includedNamespaces         string
	filepath                   string
	backupName                 string
	resourceTerminatingTimeout time.Duration
}

func main() {
	action := flag.String("a", "", "backup or restore")
	filepath := flag.String("p", "", "The location and file name of the backup file or the file to restore. Example: /Users/codegold79/kubebackups/backup.tar.gz.")
	secretName := flag.String("s", "", "Optional: The name of secret which contains remote cluster credentials")
	secretNamespace := flag.String("n", "", "Required if secret is provided: the namespace where the remote credentials secret is")
	includedNamespaces := flag.String("i", "", "Optional: included namespaces")
	flag.Parse()

	*filepath = path.Clean(*filepath)

	// TODO: make a timeout context.
	ctx := context.TODO()
	log := logrus.New()

	if err := validateCommandArgs(*action, *filepath, *secretName, *secretNamespace); err != nil {
		log.WithField("event", "validate commandline arguments").Error(err)
	}

	log.Info("obtain Kubernetes cluster credentials")
	kubeAccess, err := newKubeAccess(ctx, log, *secretName, *secretNamespace)
	if err != nil {
		log.WithField("event", "retrieve kubernetes cluster access details").Error(err)
	}

	log.Info("retrieve cluster connection clients and configs")
	clusterCxn, err := newClusterConnection(ctx, log, kubeAccess)
	if err != nil {
		log.WithField("event", "create cluster connection").Error(err)
	}

	switch *action {
	case "backup":
		config := backupConfig{
			name:               backupName,
			namespace:          namespace,
			includedNamespaces: *includedNamespaces,
			filepath:           *filepath,
		}

		log.Info("start backup")
		if err := clusterCxn.backup(ctx, log, config); err != nil {
			log.WithFields(logrus.Fields{
				"event":              "backup Kubernetes workload",
				"cluster connection": clusterCxn.host,
			})
		}
	case "restore":
		config := restoreConfig{
			name:                       restoreName,
			namespace:                  namespace,
			includedNamespaces:         "",
			filepath:                   *filepath,
			backupName:                 backupName,
			resourceTerminatingTimeout: resourceTerminatingTimeout,
		}

		log.Info("start restore")
		if err := clusterCxn.restore(ctx, log, config); err != nil {
			log.WithFields(logrus.Fields{
				"event":              "restore Kubernetes workload",
				"cluster connection": clusterCxn.host,
			})
		}
	}
}

func validateCommandArgs(action, filepath, secret, secretNS string) error {
	if action != "backup" && action != "restore" {
		return errors.New("action must be either backup or restore")
	}

	if filepath == "" {
		return errors.New("backup file location must not be empty")
	}

	if secret == "" && secretNS != "" || secret != "" && secretNS == "" {
		return errors.New("provide both remote cluster credentials secret and its namespace or neither")
	}

	return nil
}
