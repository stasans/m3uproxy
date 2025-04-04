import React, { useState, useEffect, useRef } from 'react';
import Player from './components/Player';
import Playlist from './components/Playlist';
import Overlay from './components/Overlay';
import Config from './components/Config';
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap-icons/font/bootstrap-icons.css';
import { Logger } from './utils/Logger';

function App() {
    const playlistRef = useRef(null);
    const overlayRef = useRef(null);
    const playerRef = useRef(null);

    const [showConfig, setShowConfig] = useState(false);
    const [currentChannel, setCurrentChannel] = useState(false);

    var channelNum = 0;
    var channelInputTimeout = null;
    var infoTimeout = null;

    useEffect(() => {
        const handleKeyDown = (event) => {

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
            if (event.key === 'm' || event.key === 'ColorF0Red') {
                event.preventDefault();
                setShowConfig(!showConfig);
                return;
            }

            // Key pressed:ColorF2Yellow
            // Key pressed:ColorF3Blue
            // Key pressed:MediaStop
            // Key pressed:MediaRewind
            // Key pressed:MediaFastForward

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
            if (event.key === ' ' || event.key === 'MediaPlay' || event.key === 'Pause') {
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
                overlayRef.current.setChannelNumber(channelNum);
                overlayRef.current.showChannelNumber(true);

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
        overlayRef.current.setChannelName(currentChannel ? currentChannel.name : '');
        overlayRef.current.setChannelNumber(currentChannel ? currentChannel.number + 1 : 0);
        overlayRef.current.showChannelName(true);
        overlayRef.current.showChannelNumber(true);
        playerRef.current.load(currentChannel ? currentChannel.source : '');
    }, [currentChannel]);

    const handleOnReady = () => {
        playerRef.current.load(currentChannel ? currentChannel.source : '');
    }

    const handleVideoPlay = () => {
        console.log('Video is playing');
        if (infoTimeout) {
            clearTimeout(infoTimeout);
        }
        infoTimeout = setTimeout(() => {
            overlayRef.current.hideChannelName();
            overlayRef.current.hideChannelNumber();
        }, 3000);
    }

    const handleClose = () => setShowConfig(false);
    const handleSave = () => {
        setShowConfig(false);
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
                    ref={playerRef}
                    onPlay={handleVideoPlay}
                    onReady={handleOnReady}
                />
            </div>
            <div className="overlay">
                <Overlay
                    ref={overlayRef}
                />
            </div>
        </div>
    );
}

export default App;
