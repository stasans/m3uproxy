Here's a `README.md` template for your `m3uproxy` project:

---

# m3uproxy

**m3uproxy** is a Go-based HTTP server designed to proxy HLS (HTTP Live Streaming) streams. It provides a convenient way to manage and distribute streaming streams with user management and access control features.

## Features

- **M3U Playlist Generation**: Serve dynamically generated M3U playlists for your streams.
- **XMLTV EPG Support**: Provide an XMLTV formatted Electronic Program Guide (EPG) for supported streams.
- **HLS Stream Proxying**: Securely proxy HLS streams via a token-based system.
- **User Management**: Control access to the M3U playlist and streaming streams.
- **Fallback Message**: Serve a "stream unavailable" message when necessary.

## Endpoints

### `/streams.m3u` (Restricted)
- **Description**: Returns the M3U playlist with all available streams.
- **Access**: Restricted to authenticated users.
- **Usage**: This endpoint should be accessed only after proper authentication. It provides a list of streams in the M3U format.

### `/epg.xml`
- **Description**: Returns the Electronic Program Guide (EPG) in XMLTV format.
- **Access**: Public.
- **Usage**: Useful for clients that support EPGs to fetch the program guide associated with the streams.

### `/m3uproxy/proxy/{token}/{streamId}/*`
- **Description**: Proxies the HLS stream for the specified stream.
- **Access**: Token-based access control.
- **Parameters**:
  - `token`: A unique token to authenticate the request.
  - `streamId`: The identifier of the stream.
- **Usage**: Used by clients to access the actual HLS stream. Replace `{token}` and `{streamId}` with valid values.

### `/m3uproxy/streams/*`
- **Description**: Streams a "stream unavailable" message. (FFMPEG > 5.0 must be installed)
- **Access**: Internal use only.
- **Usage**: Automatically redirected when a stream is unavailable or an error occurs.

## Installation

To install `m3uproxy`, you need to have [Go](https://golang.org/) installed on your machine.

1. Clone the repository:

   ```bash
   git clone https://github.com/a13labs/m3uproxy.git
   cd m3uproxy
   ```

2. Build the project:

   ```bash
   go build -o m3uproxy
   ```

3. Run the server:

   ```bash
   ./m3uproxy server start -m <m3u file> -e <epg file> -u <users file> -p <port>
   ```


## User Management

`m3uproxy` includes basic user management features. Users can be added, removed, or modified through a command-line interface or API.

### Adding a User

To add a user, run:

```bash
./m3uproxy users add -u <users file> <username> <password>
```

### Removing a User

To remove a user, run:

```bash
./m3uproxy users remove -u <users file> <username>
```

### Modifying a User

To modify a user's password, run:

```bash
./m3uproxy users password -u <users file> <username> <newpassword>
```

## Usage

Once the server is up and running, you can access the required endpoints as follows:

- **M3U Playlist**: `[Your Server URL]:[port]/streams.m3u`
- **EPG XML**: `[Your Server URL]:[port]/epg.xml`

Ensure that appropriate access controls are configured to restrict access to sensitive endpoints.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request or open an Issue to discuss any changes or improvements.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Thanks to the Go community for providing the tools and resources to build this project.
- Special mention to the developers of libraries and tools used in this project.
