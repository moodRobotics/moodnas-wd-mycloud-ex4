// moodd is the MoodNAS daemon: web UI, REST API, local AI and hardware loop.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/assistant"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/local"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/api"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/audit"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/auth"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/history"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/monitor"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform/sim"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/tlsutil"
)

var version = "0.1.0-dev"

func main() {
	var (
		dataDir   = flag.String("data", "/var/lib/moodnas", "state directory")
		uiDir     = flag.String("ui", "/usr/share/moodnas/ui", "built web UI directory")
		listen    = flag.String("listen", ":443", "HTTPS listen address")
		httpAddr  = flag.String("http", ":80", "HTTP listen address (redirects to HTTPS); empty to disable")
		insecure  = flag.Bool("insecure-http", false, "serve plain HTTP on -listen (development only)")
		simulate  = flag.Bool("simulate", false, "run against the simulated EX4")
		interval  = flag.Duration("interval", 30*time.Second, "monitor interval")
		mcuDevice = flag.String("mcu", "", "front-panel MCU tty (e.g. /dev/ttyS1); experimental, off by default")
		debug     = flag.Bool("debug", false, "verbose logging")
		demo      = flag.Bool("demo", false, "simulator only: create the public demo account demo / moodnas-demo")
	)
	flag.Parse()
	if *demo && !*simulate {
		fmt.Fprintln(os.Stderr, "-demo creates a publicly known password and is only allowed with -simulate")
		os.Exit(2)
	}
	demoMode = *demo

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	if err := run(log, *dataDir, *uiDir, *listen, *httpAddr, *insecure, *simulate, *interval, *mcuDevice); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, dataDir, uiDir, listen, httpAddr string, insecure, simulate bool, interval time.Duration, mcuDevice string) error {
	store, err := state.Open(dataDir)
	if err != nil {
		return fmt.Errorf("state: %w", err)
	}
	auditLog, err := audit.Open(dataDir)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	if demoMode && simulate {
		if err := createDemoAccount(store); err != nil {
			return fmt.Errorf("demo account: %w", err)
		}
		log.Warn("DEMO MODE: public account demo / moodnas-demo (simulator only)")
	}

	var p platform.Platform
	var simulator api.Simulator
	if simulate {
		s := sim.New(log)
		p, simulator = s, s
		if interval == 30*time.Second {
			interval = 2 * time.Second
		}
	} else {
		p, err = newHardware(log, mcuDevice)
		if err != nil {
			return err
		}
	}

	forecast := local.NewForecast(10*time.Minute, 6*time.Hour)
	if simulate {
		forecast = local.NewForecast(5*time.Second, time.Minute)
	}
	cfg := store.Read()
	mon := &monitor.Monitor{
		P: p, Store: store, Audit: auditLog, Log: log, Interval: interval,
		DiskModel: local.LoadDiskModel(filepath.Join(dataDir, "diskrisk-model.json")),
		Sentinel:  local.NewSentinel(),
		Forecast:  forecast,
		Fan:       local.NewFanController(cfg.Fan.QuietStart, cfg.Fan.QuietEnd),
		History:   history.New(filepath.Join(dataDir, "history.json"), interval),
	}

	srv := &api.Server{
		Store: store, Sessions: auth.NewSessions(), Limiter: auth.NewLimiter(), Audit: auditLog,
		Monitor: mon, Assistant: assistant.New(), Platform: p, Sim: simulator, Log: log,
		UIDir: uiDir, Version: version, SecureCookies: !insecure,
		// the code is on the (virtual) panel in the simulator and with the MCU driver
		PanelSetupCode: simulate || mcuDevice != "",
	}
	if code := srv.EnsureSetupCode(); code != "" {
		log.Warn("NAS not set up yet: open the web interface and enter this setup code", "setupCode", code)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go mon.Run(ctx)
	// history is saved hourly (not more: frequent writes keep disks awake) and on exit
	defer mon.History.Save()
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := mon.History.Save(); err != nil {
					log.Warn("history save", "err", err)
				}
			}
		}
	}()

	hs := &http.Server{
		Addr: listen, Handler: srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 4 * time.Minute, IdleTimeout: 2 * time.Minute,
	}
	errc := make(chan error, 2)
	if insecure {
		log.Warn("serving plain HTTP: development mode only", "addr", listen)
		go func() { errc <- hs.ListenAndServe() }()
	} else {
		cert, fp, err := tlsutil.LoadOrCreate(dataDir, cfg.Hostname)
		if err != nil {
			return fmt.Errorf("tls: %w", err)
		}
		log.Info("HTTPS certificate", "sha256", fp)
		hs.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
		ln, err := net.Listen("tcp", listen)
		if err != nil {
			return err
		}
		go func() { errc <- hs.Serve(tls.NewListener(ln, hs.TLSConfig)) }()
		if httpAddr != "" {
			go func() { errc <- http.ListenAndServe(httpAddr, redirectToHTTPS(listen)) }()
		}
	}
	log.Info("MoodNAS started", "version", version, "platform", p.Name(), "listen", listen)

	select {
	case <-ctx.Done():
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return hs.Shutdown(shut)
}

func redirectToHTTPS(listen string) http.Handler {
	_, port, _ := net.SplitHostPort(listen)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if port != "" && port != "443" {
			host = net.JoinHostPort(host, port)
		}
		http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusMovedPermanently)
	})
}
