package main

import (
	"context"
	"fmt"

	"net/url"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	"github.com/vmware/govmomi/performance"
)

// ClusterUninstaller holds the various options for the cluster we want to delete.
type ClusterUninstaller struct {
	ClusterID string
	InfraID   string

	Client     *vim25.Client
	RestClient *rest.Client

	Logger logrus.FieldLogger
}

// New returns an VSphere destroyer from ClusterMetadata.

//TODO: Use govmomi session
func new() *ClusterUninstaller {
	vim25Client, restClient, err := configureVSphereClients(context.TODO(),
		"", "", "")

	if err != nil {
		logrus.Errorln(err)
	}

	return &ClusterUninstaller{
		ClusterID:  "foo",
		InfraID:    "foo",
		Client:     vim25Client,
		RestClient: restClient,
		Logger:     logrus.StandardLogger(),
	}
}

func configureVSphereClients(ctx context.Context, vcenter, username, password string) (*vim25.Client, *rest.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	u, err := soap.ParseURL(vcenter)
	if err != nil {
		return nil, nil, err
	}
	u.User = url.UserPassword(username, password)
	c, err := govmomi.NewClient(ctx, u, true)

	if err != nil {
		return nil, nil, err
	}

	restClient := rest.NewClient(c.Client)
	err = restClient.Login(ctx, u.User)
	if err != nil {
		return nil, nil, err
	}

	return c.Client, restClient, nil
}

func main() {
	o := new()

	perfMgr := performance.NewManager(o.Client)

	finder := find.NewFinder(o.Client)
	computeResource, _ := finder.ComputeResourceList(context.TODO(), "*")

	hosts, _ := computeResource[0].Hosts(context.TODO())

	metricList, err := perfMgr.AvailableMetric(context.TODO(), hosts[0].Reference(), 20)

	if err != nil {
		spew.Dump(err)
	}

	/*
	 * Find metric list string name(s) per ManagedObject
	 * Grab objects that we are concerned with
	 */

	counterInfoByName, err := perfMgr.CounterInfoByName(context.TODO())

	if err != nil {
		spew.Dump(err)
	}
	summation := counterInfoByName["cpu.ready.summation"]

	metricListByKey := metricList.ByKey()

	summationMetric := metricListByKey[summation.Key]

	spew.Dump(summationMetric)

	spec := types.PerfQuerySpec{
		Format:     string(types.PerfFormatNormal),
		MaxSample:  int32(1),
		IntervalId: 20,
	}

	hostMobs := func() []types.ManagedObjectReference {
		var mob []types.ManagedObjectReference
		for _, h := range hosts {

			mob = append(mob, h.Reference())
		}
		return mob
	}

	metricBase, err := perfMgr.SampleByName(context.TODO(), spec, []string{"cpu.ready.summation"}, hostMobs())

	if err != nil {
		spew.Dump(err)
	}

	metricSeries, _ := perfMgr.ToMetricSeries(context.TODO(), metricBase)

	spew.Dump(metricSeries)

	for _, m := range metricSeries {
		fmt.Printf("id: %s, name: %s, value: %d\n", m.Entity.String(), m.Value[0].Name, m.Value[0].Value[0])
	}
	return
}
