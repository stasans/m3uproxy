import React, { Component, createRef } from 'react';
import {
    Player as ShakaPlayer,
    polyfill as ShakaPolyfill
} from "shaka-player/dist/shaka-player.ui";

class Player extends Component {

    constructor(props) {
        super(props);
        this.videoRef = createRef();
        this.videoContainerRef = createRef();
        this.playerRef = null;

        this.state = {
            isTV: /Philips|NETTV|SmartTvA|_TV_MT9288/i.test(navigator.userAgent),
            EMESupported: typeof window.MediaKeys !== "undefined" && typeof window.navigator.requestMediaKeySystemAccess === "function",
            isMobile: navigator.userAgent.toLowerCase().indexOf('mobile') !== -1,
            licensingUrl: '/drm/licensing',
        };

        if (__DEV__) {
            this.state.licensingUrl = `http://${window.location.hostname}:8080/drm/licensing`;
        }
    }

    componentDidMount() {

        if (this.state.EMESupported) {
            console.log('EME Supported, skipping encrypted content');
            ShakaPolyfill.installAll();
            const keys = {};

            const username = localStorage.getItem('username');
            const password = localStorage.getItem('password');
            const headers = { Authorization: 'Basic ' + btoa(`${username}:${password}`) };

            console.log('Loading licensing from server: ' + this.state.licensingUrl);
            fetch(this.state.licensingUrl, { headers })
                .then(response => response.text())
                .then(data => {
                    const response = JSON.parse(data);
                    for (const key of response.keys) {
                        if (key.kid.length !== 32 || key.k.length !== 32) {
                            console.error('Invalid key:', key);
                            continue;
                        }
                        keys[key.kid] = key.k;
                    }
                    this.playerRef.configure({
                        drm: {
                            clearKeys: keys
                        }
                    });
                })
                .catch(() => console.error('Error fetching keys from server'));
        } else {
            console.log('EME Not Supported');
        }

        this.playerRef = new ShakaPlayer(this.videoRef.current);

        // Handle player errors
        this.playerRef.addEventListener('error', (event) => {
            console.error('Shaka Player Error:', event.detail);
        });
    }

    componentDidUpdate(prevProps) {
        const { source, onLoad, onPlay } = this.props;

        if (source !== prevProps.source && this.playerRef) {
            this.playerRef.load(source).then(() => {
                if (onLoad) {
                    onLoad();
                }
                this.videoRef.current.play().then(() => {
                    if (onPlay) {
                        onPlay();
                    }
                }).catch((err) => {
                    this.handlePlayerError(err);
                });
            }).catch((err) => {
                this.handlePlayerError(err);
            });
        }
    }

    componentWillUnmount() {
        if (this.playerRef) {
            this.playerRef.destroy();
        }
    }

    handlePlayerError = (err) => {
        const { onError } = this.props;

        if (err.code === 1001) {
            console.error('DRM error:', err);
        } else if (err.code === 1002) {
            console.error('Media error:', err);
        } else if (err.code === 1004) {
            console.error('Playback error:', err);
        } else if (err.code === 1003) {
            console.error('Manifest error:', err);
        } else if (err.code === 1006) {
            console.error('Key error:', err);
        } else if (err.code === 1007) {
            console.error('License error:', err);
        } else if (err.code === 1008) {
            console.error('Network error:', err);
        } else {
            console.error('Generic error:', err);
        }
        if (onError) {
            onError(err);
        }
    };

    handleVideoClick = () => {
        if (this.videoRef.current.paused) {
            this.videoRef.current.play();
        } else {
            this.videoRef.current.pause();
        }
    };

    render() {
        return (
            <div ref={this.videoContainerRef} className="player-container">
                <video
                    ref={this.videoRef}
                    autoPlay
                    className="player"
                    onClick={this.handleVideoClick}
                />
            </div>
        );
    }
}

export default Player;