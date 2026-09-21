package relay

import (
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel/moonshot"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func ValidateMoonshotRequest(info *relaycommon.RelayInfo, request dto.Request) error {
	if info == nil || info.ApiType != constant.APITypeMoonshot {
		return nil
	}
	return moonshot.ValidateRequest(info, request)
}
