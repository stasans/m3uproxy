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
package channels

import (
	"log"
	"os"

	"github.com/a13labs/m3uproxy/cmd"
	"github.com/a13labs/m3uproxy/pkg/m3uprovider"

	"github.com/spf13/cobra"
)

var (
	configFile string
	outputFile string
	appendFile bool
)

var playlistCmd = &cobra.Command{
	Use:   "playlist",
	Short: "Generate M3U playlists",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		_, err := os.Stat(configFile)

		if os.IsNotExist(err) {
			cmd.Println("Config file not found")
			os.Exit(1)
		}

		// Load the playlist
		masterPlaylist, err := m3uprovider.LoadFromFile(configFile)
		if err != nil {
			cmd.Printf("Error loading playlist: %s\n", err)
			os.Exit(1)
		}

		log.Printf("Writing playlist to %s with %d entries.", outputFile, len(masterPlaylist.Entries))
		if outputFile == "stdout" {
			cmd.Printf("#EXTM3U")
			cmd.Printf(masterPlaylist.EntriesString())
		} else {
			content := []byte{}
			if appendFile {
				oldContent, err := os.ReadFile(outputFile)
				if err != nil {
					cmd.Println("Error reading output file")
					os.Exit(1)
				}
				content = append(content, oldContent...)
			} else {
				content = append(content, []byte("#EXTM3U\n")...)
			}
			content = append(content, []byte(masterPlaylist.EntriesString())...)
			err := os.WriteFile(outputFile, content, 0644)
			if err != nil {
				cmd.Println("Error writing to output file")
				os.Exit(1)
			}
		}
	},
}

func init() {
	cmd.RootCmd.AddCommand(playlistCmd)
	playlistCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file")
	playlistCmd.Flags().StringVarP(&outputFile, "output", "o", "streams.m3u", "Output file")
	playlistCmd.Flags().BoolVarP(&appendFile, "append", "a", false, "Append to output file")
	playlistCmd.MarkFlagRequired("config")
}
