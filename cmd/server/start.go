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

	"github.com/a13labs/m3uproxy/pkg/ffmpeg"
	"github.com/a13labs/m3uproxy/pkg/userstore"

	"github.com/spf13/cobra"
)

var (
	m3uFilePath    = "streams.m3u"
	epgFilePath    = "epg.xml"
	usersFilePath  = "users.json"
	noServiceImage = "no_service_pt.png"
	logFile        = ""
	port           = 8080
	scanTime       = 600
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the M3U proxy server",
	Long:  `Start the M3U proxy server that proxies M3U playlists and EPG data.`,
	Run: func(cmd *cobra.Command, args []string) {
		setupLogging()

		log.Printf("Starting M3U Proxy Server\n")
		log.Printf("M3U file: %s\n", m3uFilePath)
		log.Printf("EPG file: %s\n", epgFilePath)
		log.Printf("Users file: %s\n", usersFilePath)

		userstore.SetUsersFilePath(usersFilePath)
		setupStreams(scanTime)

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: setupHandlers(),
		}

		// Initialize FFmpeg
		if err := ffmpeg.Initialize(); err != nil {
			log.Fatalf("Failed to initialize FFmpeg: %v\n", err)
		}

		// Start the no service stream
		startNoServiceStream()

		// Channel to listen for termination signal (SIGINT, SIGTERM)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

		go func() {
			fmt.Printf("Starting server on :%d", port)
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
	serverCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&m3uFilePath, "m3u", "m", "streams.m3u", "Path to the M3U file (local or remote)")
	startCmd.Flags().StringVarP(&epgFilePath, "epg", "e", "epg.xml", "Path to the EPG file (local or remote)")
	startCmd.Flags().StringVarP(&usersFilePath, "users", "u", "users.json", "Path to the users JSON file")
	startCmd.Flags().StringVarP(&logFile, "logfile", "l", "", "Path to the log file (optional)")
	startCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	startCmd.Flags().IntVarP(&scanTime, "scan-time", "s", 600, "Time in seconds to scan for new streams")
	startCmd.Flags().StringVarP(&noServiceImage, "no-service-image", "i", "no_service_pt.png", "Path to the no service image")
}
