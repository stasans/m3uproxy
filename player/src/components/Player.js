import React, { useEffect, useRef } from 'react';
import shaka from 'shaka-player';

function Player({ channel }) {
    const videoRef = useRef(null);
    const playerRef = useRef(null);

    useEffect(() => {
        // Initialize Shaka Player when the component mounts
        const player = new shaka.Player(videoRef.current);
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
