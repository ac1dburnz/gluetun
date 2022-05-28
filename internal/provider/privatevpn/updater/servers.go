// Package privatevpn contains code to obtain the server information
// for the PrivateVPN provider.
package privatevpn

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qdm12/gluetun/internal/constants"
	"github.com/qdm12/gluetun/internal/models"
	"github.com/qdm12/gluetun/internal/updater/openvpn"
)

var ErrNotEnoughServers = errors.New("not enough servers found")

func (u *Updater) GetServers(ctx context.Context, minServers int) (
	servers []models.Server, err error) {
	const url = "https://privatevpn.com/client/PrivateVPN-TUN.zip"
	contents, err := u.unzipper.FetchAndExtract(ctx, url)
	if err != nil {
		return nil, err
	} else if len(contents) < minServers {
		return nil, fmt.Errorf("%w: %d and expected at least %d",
			ErrNotEnoughServers, len(contents), minServers)
	}

	countryCodes := constants.CountryCodes()

	hts := make(hostToServer)
	noHostnameServers := make([]models.Server, 0, 1) // there is only one for now

	for fileName, content := range contents {
		if !strings.HasSuffix(fileName, ".ovpn") {
			continue // not an OpenVPN file
		}

		countryCode, city, err := parseFilename(fileName)
		if err != nil {
			// treat error as warning and go to next file
			u.warner.Warn(err.Error() + " in " + fileName)
			continue
		}

		country, warning := codeToCountry(countryCode, countryCodes)
		if warning != "" {
			u.warner.Warn(warning)
		}

		host, warning, err := openvpn.ExtractHost(content)
		if warning != "" {
			u.warner.Warn(warning)
		}
		if err == nil { // found host
			hts.add(host, country, city)
			continue
		}

		ips, extractIPErr := openvpn.ExtractIPs(content)
		if warning != "" {
			u.warner.Warn(warning)
		}
		if extractIPErr != nil {
			// treat extract host error as warning and go to next file
			u.warner.Warn(extractIPErr.Error() + " in " + fileName)
			continue
		}
		server := models.Server{
			Country: country,
			City:    city,
			IPs:     ips,
			UDP:     true,
			TCP:     true,
		}
		noHostnameServers = append(noHostnameServers, server)
	}

	if len(noHostnameServers)+len(hts) < minServers {
		return nil, fmt.Errorf("%w: %d and expected at least %d",
			ErrNotEnoughServers, len(servers)+len(hts), minServers)
	}

	hosts := hts.toHostsSlice()

	hostToIPs, warnings, err := resolveHosts(ctx, u.presolver, hosts, minServers)
	for _, warning := range warnings {
		u.warner.Warn(warning)
	}
	if err != nil {
		return nil, err
	}

	hts.adaptWithIPs(hostToIPs)

	servers = hts.toServersSlice()
	servers = append(servers, noHostnameServers...)

	sortServers(servers)

	return servers, nil
}