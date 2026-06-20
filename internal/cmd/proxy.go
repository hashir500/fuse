package cmd

import (
	"os"

	"github.com/hashir500/Fuse/internal/config"
	"github.com/hashir500/Fuse/internal/proxy"
	"github.com/hashir500/Fuse/internal/spark"
	"github.com/hashir500/Fuse/internal/store"
	"github.com/spf13/cobra"
)

var proxyAddr string

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start the local Fuse proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(config.DefaultPath)
		if err != nil {
			return err
		}
		db, err := store.Open(store.DefaultDBPath)
		if err != nil {
			return err
		}
		defer db.Close()

		spark.ProxyStarted(proxyAddr)

		server := &proxy.Server{Config: cfg, Store: db, Stderr: os.Stderr}
		return server.ListenAndServe(proxyAddr)
	},
}

func init() {
	proxyCmd.Flags().StringVar(&proxyAddr, "addr", "localhost:8787", "proxy listen address")
	rootCmd.AddCommand(proxyCmd)
}
