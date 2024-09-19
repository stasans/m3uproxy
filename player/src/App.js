import React, { useState, useEffect } from 'react';
import Player from './components/Player';
import Playlist from './components/Playlist';
import Config from './components/Config';
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap-icons/font/bootstrap-icons.css';
import { Button } from 'react-bootstrap';

function App() {
    const [playlistItems, setPlaylistItems] = useState([]);
    const [selectedChannel, setSelectedChannel] = useState(null);
    const [showConfig, setShowConfig] = useState(false);

    useEffect(() => {
        // Load playlist from localStorage or show config modal
        const username = localStorage.getItem('username');
        const password = localStorage.getItem('password');
        if (username && password) {
            fetchPlaylist();
        } else {
            setShowConfig(true);
        }
    }, []);

    const fetchPlaylist = () => {
        const username = localStorage.getItem('username');
        const password = localStorage.getItem('password');
        if (username == undefined || password == undefined) {
            setShowConfig(true);
            return;
        } else {
            const headers = { Authorization: 'Basic ' + btoa(`${username}:${password}`) };

            if (process.env.NODE_ENV === 'development') {
                fetch('http://localhost:8080/streams.m3u', { headers })
                    .then(response => response.text())
                    .then(data => {
                        const items = parseM3U(data);
                        if (items.length === 0) { setShowConfig(true) } else { setPlaylistItems(items); }
                    })
                    .catch(() => setShowConfig(true));

                return;
            }

            fetch('/streams.m3u', { headers })
                .then(response => response.text())
                .then(data => {
                    const items = parseM3U(data);
                    setPlaylistItems(items);
                })
                .catch(() => setShowConfig(true));
        }
    };

    const parseM3U = (data) => {
        const lines = data.trim().split('\n');
        const items = [];
        let item = {};

        lines.forEach((line) => {
            if (line.startsWith("#EXTINF:")) {
                if (item.source) items.push(item);
                item = { tvgName: '', tvgLogo: '', source: '' };
                const tvgName = line.split(',')[1];
                item.tvgName = tvgName;
                const logoMatch = line.match(/tvg-logo="([^"]+)"/);
                if (logoMatch) item.tvgLogo = logoMatch[1];
            } else if (line && !line.startsWith("#")) {
                item.source = line;
            }
        });
        if (item.source) items.push(item);
        return items;
    };

    const handleChannelClick = (channel) => {
        setSelectedChannel(channel);
    };

    const handleShow = () => setShowConfig(true);
    const handleClose = () => setShowConfig(false);
    const handleSave = (username, password) => {
        setShowConfig(false);
        fetchPlaylist();
    }

    return (
        <div className="container-fluid">
            <div className="row d-flex justify-content-center">
                <Button onClick={handleShow}>Open Configuration</Button>
                <Config show={showConfig} onClose={handleClose} onSave={handleSave} />
                <Playlist
                    items={playlistItems}
                    onChannelClick={handleChannelClick}
                    onUpdatePlaylist={fetchPlaylist}
                />
                <div className="col-md-10">
                    <Player channel={selectedChannel} />
                </div>
            </div>
        </div>
    );
}

export default App;
