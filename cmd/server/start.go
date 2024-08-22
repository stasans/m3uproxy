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
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

var (
	m3uFilePath        = "streams.m3u"
	epgFilePath        = "epg.xml"
	usersFilePath      = "users.json"
	noServiceImage     = "no_service_pt.png"
	logFile            = ""
	noServiceAvailable = false
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

		err := loadChannels(m3uFilePath, false)
		if err != nil {
			log.Fatalf("Failed to load M3U file: %v\n", err)
			return
		}

		setupHandlers()
		log.Println("Server running on :8080")
		err = http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatalf("Failed to start server: %v\n", err)
		}
	},
}

func init() {
	serverCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&m3uFilePath, "m3u", "m", "streams.m3u", "Path to the M3U file (local or remote)")
	startCmd.Flags().StringVarP(&epgFilePath, "epg", "e", "epg.xml", "Path to the EPG file (local or remote)")
	startCmd.Flags().StringVarP(&usersFilePath, "users", "u", "users.json", "Path to the users JSON file")
	startCmd.Flags().StringVarP(&logFile, "logfile", "l", "", "Path to the log file (optional)")
	startCmd.Flags().StringVarP(&noServiceImage, "no-service-image", "i", "no_service_pt.png", "Path to the no service image")
}
