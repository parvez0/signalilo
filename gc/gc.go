package gc

import (
	"fmt"
	"time"

	"git.vshn.net/appuio/signalilo/config"
	"github.com/Nexinto/go-icinga2-client/icinga2"
)

func collectService(svc icinga2.Service, c config.Configuration) error {
	l := c.GetLogger()
	icinga := c.GetIcingaClient()

	if svc.State > 0 {
		l.V(2).Infof(fmt.Sprintf("[Collect] Skipping service %v: state=%v", svc.Name, svc.State))
		return nil
	}

	keepForNs := int64(svc.Vars["keep_for"].(float64))
	keepFor := time.Duration(keepForNs)
	lastChangeUnixNs := int64(svc.LastStateChange * 1e9)
	lastChange := time.Unix(0, lastChangeUnixNs)
	serviceAge := time.Since(lastChange)
	if serviceAge >= keepFor {
		l.V(2).Infof("[Collect] Deleting service %v: keep_for = %v; age = %v", svc.Name, keepFor, serviceAge)
		err := icinga.DeleteService(svc.FullName())
		if err != nil {
			l.Errorf(fmt.Sprintf("Error while deleting service: %v", err))
		}
	} else {
		l.V(2).Infof("[Collect] Skipping service %v: keep_for = %v; age = %v", svc.Name, keepFor, serviceAge)
	}
	return nil
}

func Collect(ts time.Time, c config.Configuration) error {
	l := c.GetLogger()
	l.Infof("[Collect] Running garbage collection at ts=%v", ts)
	// Get all signalilo services
	icinga := c.GetIcingaClient()
	services, err := icinga.ListServices()
	if err != nil {
		l.Errorf(fmt.Sprintf("[Collect] Error while listing services: %v", err))
		return err
	}
	// Iterate through services, finding ones that are managed by this
	// Signalilo and delete services which have transitioned to OK longer
	// than keep_for ago
	for _, svc := range services {
		if svc.Vars["bridge_uuid"] == c.GetConfig().UUID {
			l.Infof("[Collect] Found service %v with our bridge UUID", svc.Name)
			err = collectService(svc, c)
			if err != nil {
				l.Errorf(fmt.Sprintf("[Collect] Error garbage-collecting service: %v", err))
			}
		}
	}
	l.Infof("[Collect] Garbage collection completed in %v", time.Since(ts))
	return nil
}
