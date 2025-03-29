import React, { useEffect, useRef } from 'react';
import shaka from 'shaka-player';

function Player({ channel }) {
    const videoRef = useRef(null);
    const playerRef = useRef(null);

    useEffect(() => {
        const player = new shaka.Player();
        playerRef.current = player;

        // Attach the video element to the player
        player.attach(videoRef.current).then(() => {
            console.log('Player attached to video element');
        }).catch((err) => {
            console.error('Error attaching player to video element:', err);
        });

        if (window.globalConfig.EMESupported) {
            console.log('EME Supported, skipping encrypted content');
            shaka.polyfill.installAll();
            let clearKeysUrl = '/drm/clearkey';
            const keys = {};

            if (__DEV__) {
                clearKeysUrl = 'http://' + window.location.hostname + ':8080/drm/clearkey';
            }

            console.log('Loading clearkeys from:' + clearKeysUrl);
            fetch(clearKeysUrl, { headers })
                .then(response => response.text())
                .then(data => {
                    const response = JSON.parse(data);
                    for (const key of response.keys) {
                        if (key.kid.length !== 32 || key.k.length !== 32) {
                            console.error('Invalid key:', key);
                            continue;
                        }
                        keys[key.kid] = key.k;
                        console.log('Key:' + key.kid + ',Value:' + key.k);
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

        const username = localStorage.getItem('username');
        const password = localStorage.getItem('password');
        const headers = { Authorization: 'Basic ' + btoa(`${username}:${password}`) };

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
        if (player && channel) {
            player.load(channel.source).then(() => {
                console.log('Video loaded successfully');
            }).catch((err) => {
                console.error('Error loading source ' + channel.source, err);
            });
        }
    }, [channel]);

    return (
        <div>
            <video ref={videoRef} controls autoPlay className="w-100 player" />
        </div>
    );
}

export default Player;