/*
Copyright © 2024 Alexandre Pires

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a13labs/m3uproxy/cmd"
	rootCmd "github.com/a13labs/m3uproxy/cmd"
	"github.com/a13labs/m3uproxy/pkg/ffmpeg"
	"github.com/a13labs/m3uproxy/pkg/userstore"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the M3U proxy server",
	Long:  `Start the M3U proxy server that proxies M3U playlists and EPG data.`,
	Run: func(cmd *cobra.Command, args []string) {

		config, err := rootCmd.LoadConfig()
		if err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}

		log.Printf("Starting M3U Proxy Server\n")
		log.Printf("Playlist: %s\n", config.Playlist)
		log.Printf("EPG: %s\n", config.Epg)
		log.Printf("No Service: %s\n", config.NoServiceImage)
		log.Printf("Auth Provider: %s\n", config.Auth.Provider)
		log.Printf("Port: %d\n", config.Port)

		err = userstore.InitializeAuthProvider(config.Auth.Provider, config.Auth.Settings)
		if err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}

		setupLogging(config)
		setupStreams(config)

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", config.Port),
			Handler: setupHandlers(config),
		}

		// Initialize FFmpeg
		if err := ffmpeg.Initialize(); err != nil {
			log.Fatalf("Failed to initialize FFmpeg: %v\n", err)
		}

		// Start the no service stream
		startNoServiceStream(config)

		// Channel to listen for termination signal (SIGINT, SIGTERM)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		go func() {
			fmt.Printf("Starting server on :%d", config.Port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Println("Server failed:", err)
			}
		}()

		<-quit // Wait for SIGINT or SIGTERM

		fmt.Println("Shutting down server...")

		// Stop the no service stream
		stopNoServiceStream()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			fmt.Println("Server forced to shutdown:", err)
		}

		fmt.Println("Server exiting")
	},
}

func init() {
	cmd.RootCmd.AddCommand(serverCmd)
}
