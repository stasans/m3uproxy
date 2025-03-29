import React, { useState, useEffect } from 'react';
import Player from './components/Player';
import Playlist from './components/Playlist';
import Config from './components/Config';
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap-icons/font/bootstrap-icons.css';
import { Button } from 'react-bootstrap';

window.globalConfig = {
    isTV: /Philips|NETTV|SmartTvA|_TV_MT9288/i.test(navigator.userAgent),
    EMESupported: typeof window.MediaKeys !== "undefined" && typeof window.navigator.requestMediaKeySystemAccess === "function",
    isMobile: navigator.userAgent.toLowerCase().indexOf('mobile') !== -1,
};

if (__DEV__) {

    if (window.globalConfig.isTV) {
        console.log('Development mode: TV detected');
        function logError(error) {
            const img = new Image();
            img.src = "http://" + window.location.hostname + ":3000/log?msg=" + encodeURIComponent(error);
        }

        window.onerror = function (msg, url, lineNo, columnNo, error) {
            logError(`Error: ${msg} at ${url}:${lineNo}:${columnNo}`);
        };

        console.log = (msg) => logError("LOG: " + msg);
        console.error = (msg) => logError("ERROR: " + msg);
        console.warn = (msg) => logError("WARNING: " + msg);
    }

    console.log('Development mode: Logging enabled');
    // log globalConfig
    console.log('Global Config:');
    for (const [key, value] of Object.entries(window.globalConfig)) {
        console.log(`- ${key}: ${value}`);
    }
}

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
            // Fetch playlist periodically every 5 minutes
            const intervalId = setInterval(fetchPlaylist, 5 * 60 * 1000);
            return () => clearInterval(intervalId); // Cleanup interval on unmount
        } else {
            setShowConfig(true);
        }
    }, []);

    useEffect(() => {
        const handleKeyDown = (event) => {
            // Page up/down key handling
            if (event.key === 'PageUp' || event.key === 'PageDown') {
                event.preventDefault();
                const currentChannelIndex = parseInt(localStorage.getItem('current_channel_index')) || 0;
                const channelList = window.globalConfig.channelList || [];
                let newChannelIndex = currentChannelIndex;
                console.log('Current channel index:' + currentChannelIndex);

                if (event.key === 'PageUp') {
                    newChannelIndex++;
                } else if (event.key === 'PageDown') {
                    newChannelIndex--;
                }
                // Wrap around the channel list
                if (newChannelIndex >= channelList.length) {
                    newChannelIndex = 0;
                }
                if (newChannelIndex < 0) {
                    newChannelIndex = channelList.length - 1;
                }

                // Update the current channel in localStorage
                localStorage.setItem('current_channel_index', newChannelIndex);
                console.log('Current channel index:' + newChannelIndex);
                handleChannelClick(channelList[newChannelIndex]);
            }
            // M show/hide config
            if (event.key === 'm') {
                event.preventDefault();
                setShowConfig(!showConfig);
            }
        };

        // Add the keydown event listener
        window.addEventListener('keydown', handleKeyDown);

        return () => {
            // Clean up the event listener on unmount
            window.removeEventListener('keydown', handleKeyDown);
        };
    }, []);

    const fetchPlaylist = () => {
        const username = localStorage.getItem('username');
        const password = localStorage.getItem('password');
        if (!username || !password) {
            setShowConfig(true);
            return;
        }

        let streamsUrl = '/streams.m3u';
        if (__DEV__) {
            streamsUrl = `http://${window.location.hostname}:8080/streams.m3u`;
            console.log('Loading streams from ' + streamsUrl);
        }
        const headers = { Authorization: 'Basic ' + btoa(`${username}:${password}`) };

        fetch(streamsUrl, { headers })
            .then(response => response.text())
            .then(data => {
                const items = parseM3U(data);
                setPlaylistItems(items);
                window.globalConfig.channelList = items;
                let previousChannelIndex = parseInt(localStorage.getItem('current_channel_index')) || 0;
                if (previousChannelIndex >= items.length) {
                    previousChannelIndex = 0;
                }
                if (previousChannelIndex < 0) {
                    previousChannelIndex = items.length - 1;
                }
                const currentChannel = items[previousChannelIndex];
                setSelectedChannel(currentChannel);
                console.log('Current channel index:' + previousChannelIndex);
            })
            .catch((error) => {
                console.error('Error fetching playlist:' + error);
                setPlaylistItems([]);
                setSelectedChannel(null);
            }
            );

    };

    const parseM3U = (data) => {
        const lines = data.trim().split('\n');
        const items = [];
        let item = {};

        let channel_num = 0;
        lines.forEach((line) => {
            if (line.startsWith("#EXTINF:")) {
                if (item.source) items.push(item);
                item = { tvgName: '', tvgLogo: '', source: '', channel_num: channel_num++ };
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
        localStorage.setItem('current_channel_index', channel.channel_num);
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
                <Config show={showConfig} onClose={handleClose} onSave={handleSave} />
                <div className="col-sm-2 sidebar">
                    <Playlist
                        items={playlistItems}
                        onChannelClick={handleChannelClick}
                        onUpdatePlaylist={fetchPlaylist}
                    >
                    </Playlist>
                </div>
                <div className="col-sm-10 content">
                    <div className="d-flex justify-content-between align-items-center toolbar">
                        <span id="channel_title" className="mb-0"></span>
                    </div>
                    <Player channel={selectedChannel} />
                </div>
            </div>
        </div>
    );
}

export default App;
