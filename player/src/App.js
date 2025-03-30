import React, { useState, useEffect, useRef } from 'react';
import Player from './components/Player';
import Playlist from './components/Playlist';
import Config from './components/Config';
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap-icons/font/bootstrap-icons.css';
import { Logger } from './utils/logger';

function App() {
    const channelNameRef = useRef(null);
    const channelNumberRef = useRef(null);
    const playlistRef = useRef(null);
    const [showConfig, setShowConfig] = useState(false);
    const [channelNameVisible, setChannelNameVisible] = useState(false);
    const [channelNumberVisible, setChannelNumberVisible] = useState(false);
    const [currentChannel, setCurrentChannel] = useState(false);

    var channelNum = 0;
    var channelInputTimeout = null;
    var infoTimeout = null;

    useEffect(() => {
        const handleKeyDown = (event) => {

            Logger.info('Key pressed:', event.key);
            // Page up/down key handling
            if (event.key === 'PageUp') {
                event.preventDefault();
                playlistRef.current.channelUp()
                return;
            }

            if (event.key === 'PageDown') {
                event.preventDefault();
                playlistRef.current.channelDown()
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
                        Logger.info('Already in fullscreen mode');
                        return;
                    }
                    // Request fullscreen on the document element and set video width to 100%
                    document.documentElement.requestFullscreen().then(() => {
                        Logger.info('Entered fullscreen mode');
                    }
                    ).catch((error) => {
                        Logger.error('Error entering fullscreen:', error);
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

                if (channelInputTimeout) {
                    clearTimeout(channelInputTimeout);
                }

                channelInputTimeout = setTimeout(() => {
                    // cancel previous timeout
                    const newChannelNum = channelNum - 1;
                    channelNum = 0;
                    playlistRef.current.changeChannel(newChannelNum);
                    clearTimeout(channelInputTimeout);
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

    useEffect(() => {
        setChannelNameVisible(true);
        setChannelNumberVisible(true);
    }, [currentChannel]);

    const handleVideoPlay = () => {
        if (infoTimeout) {
            clearTimeout(infoTimeout);
        }
        infoTimeout = setTimeout(() => {
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
                    ref={playlistRef}
                    setCurrentChannel={setCurrentChannel}
                >
                </Playlist>
            </div>
            <div className="content">
                <Player
                    source={currentChannel ? currentChannel.source : ''}
                    onPlay={handleVideoPlay}
                />
                <div ref={channelNameRef} className="channel-name" style={{
                    opacity: channelNameVisible ? 1 : 0,
                    transition: channelNameVisible ? "" : "opacity 2s ease-out"
                }} >{currentChannel ? currentChannel.tvgName : ""}</div>
                <div ref={channelNumberRef} className="channel-number" style={{
                    opacity: channelNumberVisible ? 1 : 0,
                    transition: channelNumberVisible ? "" : "opacity 2s ease-out"
                }} >{currentChannel ? currentChannel.channel_num + 1 : 0}</div>
            </div>
        </div>
    );
}

export default App;
