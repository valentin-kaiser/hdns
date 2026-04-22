package web

import (
	"crypto/tls"
	"crypto/x509/pkix"
	"embed"
	"io/fs"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/valentin-kaiser/go-core/apperror"
	"github.com/valentin-kaiser/go-core/interruption"
	"github.com/valentin-kaiser/go-core/logging/log"
	"github.com/valentin-kaiser/go-core/security"
	"github.com/valentin-kaiser/go-core/web"
	"github.com/valentin-kaiser/hdns/pkg/config"
	"github.com/valentin-kaiser/hdns/pkg/proto/service"
	"github.com/valentin-kaiser/hdns/pkg/web/api"
)

//go:embed static
var static embed.FS

func Start() {
	defer interruption.Catch()

	frontend, err := fs.Sub(static, "static")
	if err != nil {
		log.Error().Err(err).Msg("failed to create frontend file system")
		return
	}

	c, err := security.LoadCertAndConfig(config.Get().CertificatePath, config.Get().KeyPath, "", tls.NoClientCert)
	if err != nil {
		log.Warn().Err(err).Msg("failed to load TLS certificate and key, generating self-signed certificate")
		cert, _, err := security.GenerateSelfSignedCertificate(pkix.Name{
			CommonName: "HDNS",
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to generate self-signed certificate")
			return
		}

		err = security.WriteCertificate(cert, config.Get().CertificatePath, config.Get().KeyPath)
		if err != nil {
			log.Error().Err(err).Msg("failed to write self-signed certificate and key to disk")
			return
		}

		c, err = security.LoadCertAndConfig(config.Get().CertificatePath, config.Get().KeyPath, "", tls.NoClientCert)
		if err != nil {
			log.Error().Err(err).Msg("failed to load self-signed TLS certificate and key")
			return
		}
	}

	done := make(chan error, 1)
	s := web.Instance().
		WithTLS(c).
		WithPort(uint16(config.Get().WebPort), web.ProtocolHTTPS).
		WithSecurityHeaders().
		WithCORSHeaders(&web.CORSConfig{
			AllowOrigin:  "*",
			AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodOptions},
			AllowHeaders: []string{"Content-Type"},
			MaxAge:       3600,
		}).
		WithGzip().
		WithLog().
		WithFS([]string{"/"}, frontend).
		WithWebsocket("/ws", func(w http.ResponseWriter, r *http.Request, conn *websocket.Conn) {
			defer apperror.Catch(conn.Close, "failed to close websocket connection")
		})

	s.WithJRPC("/rpc", service.RegisterHDNSServer(&api.Server{}))

	s.StartAsync(done)

	err = <-done
	if err != nil {
		log.Error().Err(err).Msg("web server failed")
	}
}

func Restart() {
	done := make(chan error, 1)
	web.Instance().
		WithPort(uint16(config.Get().WebPort), web.ProtocolHTTPS).
		RestartAsync(done)
	err := <-done
	if err != nil {
		log.Error().Err(err).Msg("web server failed to restart")
	}
}

func Stop() error {
	defer interruption.Catch()
	err := web.Instance().Stop()
	if err != nil {
		return apperror.NewError("failed to stop web server").AddError(err)
	}
	return nil
}
