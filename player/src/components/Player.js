import React, { useEffect, useRef } from 'react';
import {
    Player as ShakaPlayer,
    polyfill as ShakaPolyfill
} from "shaka-player/dist/shaka-player.ui";

function Player({ source, onLoad, onPlay, onError }) {
    const videoRef = useRef(null);
    const videoContainerRef = useRef(null);
    const playerRef = useRef(null);

    useEffect(() => {
        if (window.globalConfig.EMESupported) {
            console.log('EME Supported, skipping encrypted content');
            ShakaPolyfill.installAll();
            const keys = {};

            const username = localStorage.getItem('username');
            const password = localStorage.getItem('password');
            const headers = { Authorization: 'Basic ' + btoa(`${username}:${password}`) };

            console.log('Loading licensing from server');
            fetch(window.globalConfig.licensingUrl, { headers })
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
                    player.configure({
                        drm: {
                            clearKeys: keys
                        }
                    });
                })
                .catch(() => console.error('Error fetching keys from server'));
        } else {
            console.log('EME Not Supported');
        }

        const player = new ShakaPlayer(videoRef.current);
        playerRef.current = player;

        // Handle player errors
        player.addEventListener('error', (event) => {
            console.error('Shaka Player Error:', event.detail);
        });

        // Cleanup player on component unmount
        return () => {
            if (playerRef.current) {
                playerRef.current.destroy();
            }
        };
    }, []);

    useEffect(() => {
        // Load the video when the channel prop changes
        const player = playerRef.current;
        if (player) {
            player.load(source).then(() => {
                onLoad();
                videoRef.current.play().then(() => {
                    onPlay();
                }
                ).catch((err) => {
                    if (err.code === 1001) {
                        console.error('DRM error:', err);
                        onError(err);
                    } else if (err.code === 1002) {
                        console.error('Media error:', err);
                        onError(err);
                    } else if (err.code === 1004) {
                        console.error('Playback error:', err);
                        onError(err);
                    } else if (err.code === 1003) {
                        console.error('Manifest error:', err);
                        onError(err);
                    } else if (err.code === 1006) {
                        console.error('Key error:', err);
                        onError(err);
                    } else if (err.code === 1007) {
                        console.error('License error:', err);
                        onError(err);
                    } else if (err.code === 1008) {
                        console.error('Network error:', err);
                        onError(err);
                    } else {
                        console.error('Generic error:', err);
                    }
                });
            }).catch((err) => {
                if (err.code === 1001) {
                    console.error('DRM error:', err);
                    onError(err);
                } else if (err.code === 1002) {
                    console.error('Media error:', err);
                    onError(err);
                } else if (err.code === 1004) {
                    console.error('Playback error:', err);
                    onError(err);
                } else if (err.code === 1003) {
                    console.error('Manifest error:', err);
                    onError(err);
                } else if (err.code === 1006) {
                    console.error('Key error:', err);
                    onError(err);
                } else if (err.code === 1007) {
                    console.error('License error:', err);
                    onError(err);
                } else if (err.code === 1008) {
                    console.error('Network error:', err);
                    onError(err);
                } else {
                    console.error('Generic error:', err);
                }

            });
        }
    }, [source]);

    return (
        <div ref={videoContainerRef} className="player-container">
            <video ref={videoRef} autoPlay className="player"
                onClick={() => {
                    if (videoRef.current.paused) {
                        videoRef.current.play();
                    } else {
                        videoRef.current.pause();
                    }
                }
                }
            />
        </div>
    );
}

export default Player;