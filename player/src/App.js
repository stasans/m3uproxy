import React, { Component, createRef } from 'react';
import Player from './components/Player';
import Playlist from './components/Playlist';
import Overlay from './components/Overlay';
import Config from './components/Config';
import 'bootstrap/dist/css/bootstrap.min.css';
import 'bootstrap-icons/font/bootstrap-icons.css';
import { Logger } from './utils/Logger';

class App extends Component {
    constructor(props) {
        super(props);

        this.playlistRef = createRef();
        this.overlayRef = createRef();
        this.playerRef = createRef();

        this.state = {
            showConfig: false,
            currentChannel: null,
        };

        this.channelNum = 0;
        this.channelInputTimeout = null;
        this.infoTimeout = null;
    }

    componentDidMount() {
        window.addEventListener('keydown', this.handleKeyDown);
    }

    componentWillUnmount() {
        window.removeEventListener('keydown', this.handleKeyDown);
    }

    handleKeyDown = (event) => {
        const { showConfig } = this.state;

        // Page up/down key handling
        if (event.key === 'PageUp') {
            event.preventDefault();
            this.playlistRef.current.channelUp();
            return;
        }

        if (event.key === 'PageDown') {
            event.preventDefault();
            this.playlistRef.current.channelDown();
            return;
        }

        // M show/hide config
        if (event.key === 'm' || event.key === 'ColorF0Red') {
            event.preventDefault();
            window.removeEventListener('keydown', this.handleKeyDown);
            this.setState({ showConfig: !showConfig });
            return;
        }

        // F key to toggle fullscreen
        if (event.key === 'f') {
            if (document.fullscreenElement) {
                document.exitFullscreen();
            } else {
                if (document.fullscreenElement) {
                    Logger.info('Already in fullscreen mode');
                    return;
                }
                document.documentElement.requestFullscreen()
                    .then(() => Logger.info('Entered fullscreen mode'))
                    .catch((error) => Logger.error('Error entering fullscreen:', error));
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

        // Handle digit keys for channel input
        if (event.key >= '0' && event.key <= '9') {
            event.preventDefault();
            const digit = parseInt(event.key, 10);
            this.channelNum = this.channelNum * 10 + digit;
            this.overlayRef.current.setChannelNumber(this.channelNum);
            this.overlayRef.current.showChannelNumber(true);

            if (this.channelInputTimeout) {
                clearTimeout(this.channelInputTimeout);
            }

            this.channelInputTimeout = setTimeout(() => {
                const newChannelNum = this.channelNum - 1;
                this.channelNum = 0;
                this.playlistRef.current.changeChannel(newChannelNum);
                clearTimeout(this.channelInputTimeout);
            }, 3000);
        }
    };

    setCurrentChannel = (channel) => {

        this.setState({ currentChannel: channel });
        this.channelNum = channel.number + 1;
        this.overlayRef.current.setChannelName(channel.name);
        this.overlayRef.current.setChannelNumber(channel.number + 1);
        this.overlayRef.current.showChannelName(true);
        this.overlayRef.current.showChannelNumber(true);
        this.playerRef.current.load(channel.source);
    }

    handleOnReady = () => {
        const { currentChannel } = this.state;
        this.playerRef.current.load(currentChannel ? currentChannel.source : '');
    };

    handleVideoPlay = () => {
        console.log('Video is playing');
        if (this.infoTimeout) {
            clearTimeout(this.infoTimeout);
        }
        this.infoTimeout = setTimeout(() => {
            this.overlayRef.current.hideChannelName();
            this.overlayRef.current.hideChannelNumber();
        }, 3000);
    };

    handleConfigClose = () => {
        window.addEventListener('keydown', this.handleKeyDown);
        this.setState({ showConfig: false });
    }

    render() {
        const { showConfig } = this.state;

        return (
            <div className="app">
                <Config show={showConfig} onClose={this.handleConfigClose} onSave={this.handleSave} />
                <div className="sidebar">
                    <Playlist
                        ref={this.playlistRef}
                        setCurrentChannel={this.setCurrentChannel}
                    />
                </div>
                <div className="content">
                    <Player
                        ref={this.playerRef}
                        onPlay={this.handleVideoPlay}
                        onReady={this.handleOnReady}
                    />
                </div>
                <div className="overlay">
                    <Overlay ref={this.overlayRef} />
                </div>
            </div>
        );
    }
}

export default App;
