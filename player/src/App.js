import React, { useState, useEffect, useRef } from 'react';
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
    channelsUrl: '/streams.m3u',
    licensingUrl: '/drm/licensing',
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

    window.globalConfig.channelsUrl = `http://${window.location.hostname}:8080${window.globalConfig.channelsUrl}`;
    window.globalConfig.licensingUrl = `http://${window.location.hostname}:8080${window.globalConfig.licensingUrl}`;

    console.log('Development mode: Logging enabled');
    console.log('Global Config:');
    for (const [key, value] of Object.entries(window.globalConfig)) {
        console.log(`- ${key}: ${value}`);
    }

}

function App() {
    const [playlistItems, setPlaylistItems] = useState([]);
    const [showConfig, setShowConfig] = useState(false);
    const channelNameRef = useRef(null);
    const channelNumberRef = useRef(null);
    const [channelNameVisible, setChannelNameVisible] = useState(false);
    const [channelNumberVisible, setChannelNumberVisible] = useState(false);
    const [currentChannel, setCurrentChannel] = useState(false);
    var channelNum = 0;

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

            console.log('Key pressed:', event.key);
            // Page up/down key handling
            if (event.key === 'PageUp' || event.key === 'PageDown') {
                event.preventDefault();
                let currentChannelIndex = parseInt(localStorage.getItem('current_channel_index')) || 0;
                const channelList = window.globalConfig.channelList || [];

                if (channelList.length === 0) {
                    console.error('No channels available');
                    return;
                }

                if (event.key === 'PageUp') {
                    currentChannelIndex++;
                } else if (event.key === 'PageDown') {
                    currentChannelIndex--;
                }
                // Wrap around the channel list
                if (currentChannelIndex >= channelList.length) {
                    currentChannelIndex = 0;
                }
                if (currentChannelIndex < 0) {
                    currentChannelIndex = channelList.length - 1;
                }

                // Update the current channel in localStorage
                requestChannelChange(channelList[currentChannelIndex]);
                return;
            }

            // M show/hide config
            if (event.key === 'm') {
                event.preventDefault();
                setShowConfig(!showConfig);
                return;
            }

            // F key to toggle fullscreen
            if (event.key === 'f') {
                if (document.fullscreenElement) {
                    document.exitFullscreen();
                } else {
                    // Check if the document is already in fullscreen mode
                    if (document.fullscreenElement) {
                        console.log('Already in fullscreen mode');
                        return;
                    }
                    // Request fullscreen on the document element and set video width to 100%
                    document.documentElement.requestFullscreen().then(() => {
                        console.log('Entered fullscreen mode');
                    }
                    ).catch((error) => {
                        console.error('Error entering fullscreen:', error);
                    });
                }
            }
            // Escape key to exit fullscreen
            if (event.key === 'Escape') {
                if (document.fullscreenElement) {
                    document.exitFullscreen();
                }
            }

            // Space key to pause/play
            if (event.key === ' ') {
                event.preventDefault();
                const video = document.querySelector('video');
                if (video) {
                    if (video.paused) {
                        video.play();
                    } else {
                        video.pause();
                    }
                }
            }

            // When pressing a digit key (0-9), the input will be read for 3 seconds, each subsequent digit will be used to compute the channel number
            // and the channel will be changed to the corresponding channel number
            if (event.key >= '0' && event.key <= '9') {
                event.preventDefault();
                const digit = parseInt(event.key, 10);
                channelNum = channelNum * 10 + digit;
                channelNumberRef.current.innerText = channelNum;
                setChannelNumberVisible(true);

                if (window.globalConfig.changeTimeout) {
                    clearTimeout(window.globalConfig.changeTimeout);
                }
                window.globalConfig.changeTimeout = setTimeout(() => {
                    // cancel previous timeout
                    const channelList = window.globalConfig.channelList || [];
                    const newChannelNum = channelNum - 1;
                    channelNum = 0;

                    if (newChannelNum >= channelList.length) {
                        console.error('Channel number out of range');
                        return;
                    }

                    // Wrap around the channel list
                    requestChannelChange(channelList[newChannelNum]);
                }, 3000);
                return;
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
        const headers = { Authorization: 'Basic ' + btoa(`${username}:${password}`) };

        // const headers = buildRequestHeaders();
        console.log('Fetching playlist from URL:', window.globalConfig.channelsUrl);
        fetch(window.globalConfig.channelsUrl, { headers })
            .then(response => response.text())
            .then(data => {
                const items = parsePlaylist(data);
                if (items.length === 0) {
                    console.error('No channels available');
                    window.globalConfig.channelList = [];
                    setPlaylistItems([]);
                    setCurrentChannel(null);
                    return;
                }

                window.globalConfig.channelList = items;
                setPlaylistItems(items);

                // Set the selected channel to the last watched channel
                let previousChannelIndex = parseInt(localStorage.getItem('current_channel_index')) || 0;
                if (previousChannelIndex >= items.length) {
                    previousChannelIndex = 0;
                }
                if (previousChannelIndex < 0) {
                    previousChannelIndex = items.length - 1;
                }
                setCurrentChannel(items[previousChannelIndex]);
            })
            .catch((error) => {
                console.error('Error fetching playlist:' + error);
                window.globalConfig.channelList = [];
                setPlaylistItems([]);
                setCurrentChannel(null);
            }
            );

    };

    const parsePlaylist = (data) => {
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

    const requestChannelChange = (channel) => {
        localStorage.setItem('current_channel_index', channel.channel_num);
        setChannelNameVisible(true);
        setChannelNumberVisible(true);
        setCurrentChannel(channel);
    };

    const handleVideoError = (error) => {
        console.error('Video error:', error);
        channelNameRef.current.innerText = currentChannel.tvgName + ' - channel not available';
        setChannelNameVisible(true);
        setChannelNumberVisible(true);
    }
    const handleVideoLoad = () => {
    }

    const handleVideoPlay = () => {
        setTimeout(() => {
            setChannelNameVisible(false);
            setChannelNumberVisible(false);
        }, 3000);
    }

    const handleClose = () => setShowConfig(false);
    const handleSave = () => {
        setShowConfig(false);
        fetchPlaylist();
    }

    return (
        <div className="app">
            <Config show={showConfig} onClose={handleClose} onSave={handleSave} />
            <div className="sidebar">
                <Playlist
                    items={playlistItems}
                    onChannelClick={requestChannelChange}
                    onUpdatePlaylist={fetchPlaylist}
                >
                </Playlist>
            </div>
            <div className="content">
                <Player
                    source={currentChannel ? currentChannel.source : ''}
                    onError={handleVideoError}
                    onLoad={handleVideoLoad}
                    onPlay={handleVideoPlay}
                />
                <div ref={channelNameRef} className="channel-name" style={{
                    opacity: channelNameVisible ? 1 : 0,
                    transition: channelNameVisible ? "" : "opacity 2s ease-out"
                }} >{currentChannel.tvgName}</div>
                <div ref={channelNumberRef} className="channel-number" style={{
                    opacity: channelNumberVisible ? 1 : 0,
                    transition: channelNumberVisible ? "" : "opacity 2s ease-out"
                }} >{currentChannel.channel_num + 1}</div>
            </div>
        </div>
    );
}

export default App;
