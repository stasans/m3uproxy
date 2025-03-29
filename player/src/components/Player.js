import React, { useEffect, useRef, useState } from 'react';
import 'shaka-player/dist/controls.css';
const shaka = require('shaka-player/dist/shaka-player.ui.js');

function Player({ channel_num }) {
    const videoRef = useRef(null);
    const videoContainerRef = useRef(null);
    const playerRef = useRef(null);
    const [channel, setChannel] = useState(null);
    const [fadeOut, setFadeOut] = useState(false);

    useEffect(() => {
        const player = new shaka.Player();
        playerRef.current = player;

        // Attach the video element to the player
        player.attach(videoRef.current).then(() => {
            // const video = videoRef.current;
            // const videoContainer = videoContainerRef.current;
            // const ui = new shaka.ui.Overlay(player, videoContainer, video);
            // ui.configure({
            //     controlPanelElements: ['play_pause', 'spacer'],
            //     addSeekBar: true,
            // });

        }).catch((err) => {
            console.error('Error attaching player to video element:', err);
        });

        if (window.globalConfig.EMESupported) {
            console.log('EME Supported, skipping encrypted content');
            shaka.polyfill.installAll();
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
        console.log('Loading channel:', channel_num);
        const player = playerRef.current;
        if (player && window.globalConfig.channelList) {
            const channel = window.globalConfig.channelList[channel_num];
            if (!channel) {
                console.error('Channel not found');
                return;
            }
            setChannel(channel);
            setFadeOut(false);
            player.load(channel.source).then(() => {
                console.log('Video loaded successfully');
                videoRef.current.play().catch((err) => {
                    console.error('Error starting playback:', err);
                });
                setFadeOut(false);
                setTimeout(() => {
                    setFadeOut(true);
                }, 3000); // Hide overlay after 2 seconds
            }).catch((err) => {
                console.error('Error loading source ' + channel.source, err);
            });
        }
    }, [channel_num]);

    return (
        <div id="video-container" ref={videoContainerRef} className="player-container">
            <video id="video" ref={videoRef} autoPlay controls className="w-100 player" />
            {channel && (
                <>
                    <div className="channel-name" style={{
                        opacity: fadeOut ? 0 : 1,
                        transition: fadeOut ? "opacity 2s ease-out" : ""
                    }} >{channel.tvgName}</div>
                    <div className="channel-number" style={{
                        opacity: fadeOut ? 0 : 1,
                        transition: fadeOut ? "opacity 2s ease-out" : ""
                    }} >{channel_num}</div>
                </>
            )}
        </div>
    );
}

export default Player;