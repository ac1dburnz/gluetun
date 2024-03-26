package env

import (
	"fmt"
	"strings"

	"github.com/qdm12/gluetun/internal/configuration/settings"
	"github.com/qdm12/gluetun/internal/constants/providers"
	"github.com/qdm12/gluetun/internal/constants/vpn"
	"github.com/qdm12/gosettings/sources/env"
)

func (s *Source) readProvider(vpnType string) (provider settings.Provider, err error) {
	provider.Name = s.readVPNServiceProvider(vpnType)
	var providerName string
	if provider.Name != nil {
		providerName = *provider.Name
	}

	provider.ServerSelection, err = s.readServerSelection(providerName, vpnType)
	if err != nil {
		return provider, fmt.Errorf("server selection: %w", err)
	}

	provider.PortForwarding, err = s.readPortForward()
	if err != nil {
		return provider, fmt.Errorf("port forwarding: %w", err)
	}

	return provider, nil
}

func (s *Source) readVPNServiceProvider(vpnType string) (vpnProviderPtr *string) {
	valuePtr := s.env.Get("VPN_SERVICE_PROVIDER", env.RetroKeys("VPNSP"))
	if valuePtr == nil {
		if vpnType != vpn.Wireguard && s.env.Get("OPENVPN_CUSTOM_CONFIG") != nil {
			// retro compatibility
			return ptrTo(providers.Custom)
		}
		return nil
	}

	value := *valuePtr
	value = strings.ToLower(value)
	if value == "pia" { // retro compatibility
		return ptrTo(providers.PrivateInternetAccess)
	}

	return ptrTo(value)
}
