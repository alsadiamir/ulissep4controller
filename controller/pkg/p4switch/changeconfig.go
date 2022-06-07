package p4switch

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

)



func changeConfig(ctx context.Context, sw *GrpcSwitch, configName string) {
	if err := sw.ChangeConfig(configName); err != nil {
		if status.Convert(err).Code() == codes.Canceled {
			sw.GetLogger().Warn("Failed to update config, restarting")
			if err := sw.RunSwitch(ctx); err != nil {
				sw.GetLogger().Errorf("Cannot start")
				sw.GetLogger().Errorf("%v", err)
			}
		} else {
			sw.GetLogger().Errorf("Error updating swConfig: %v", err)
		}
		return
	}
	sw.GetLogger().Tracef("Config updated to %s, ", configName)
}