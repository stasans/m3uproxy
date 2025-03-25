import React, { useEffect, useRef } from 'react';
import shaka from 'shaka-player';

function Player({ channel }) {
    const videoRef = useRef(null);
    const playerRef = useRef(null);

    useEffect(() => {
        shaka.polyfill.installAll();
        // Initialize Shaka Player when the component mounts
        const player = new shaka.Player(videoRef.current);

        const username = localStorage.getItem('username');
        const password = localStorage.getItem('password');
        const headers = { Authorization: 'Basic ' + btoa(`${username}:${password}`) };
        var clearKeysUrl = '/drm/clearkey';
        const keys = {};

        // Load clearkeys from the server, for now we are hardcoding them
        if (__DEV__) {
            clearKeysUrl = 'http://localhost:8080/drm/clearkey';
        }

        fetch(clearKeysUrl, { headers })
            .then(response => response.text())
            .then(data => {
                const response = JSON.parse(data);
                for (const key of response.keys) {
                    keys[key.kid] = key.k;
                }
                player.configure({
                    drm: {
                        clearKeys: keys
                    }
                });
            })
            .catch(() => console.error('Error fetching clearkeys'));

        playerRef.current = player;

        // Handle player errors
        player.addEventListener('error', (event) => {
            console.error('Shaka Player Error:', event.detail);
        });


        // const previousChannel = localStorage.getItem('lastChannel');
        // if (previousChannel) {
        //     console.log('Previous channel:', previousChannel);
        //     const channel = JSON.parse(previousChannel);
        //     player.load(channel.source).catch((err) => {
        //         console.error('Error loading video:', err);
        //     });
        // }

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
            player.load(channel.source).catch((err) => {
                console.error('Error loading video:', err);
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
