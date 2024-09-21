import React, { useState, useRef, useEffect } from 'react';

function Playlist({ items, onChannelClick }) {
    const [searchTerm, setSearchTerm] = useState('');
    const channelsRef = useRef(null);
    const topTriggerRef = useRef(null);
    const bottomTriggerRef = useRef(null);

    const filteredItems = items.filter(item =>
        item.tvgName.toLowerCase().includes(searchTerm.toLowerCase())
    );

    const [images, setImages] = useState({});

    const loadImage = (url) => {
        return new Promise((resolve, reject) => {
            const img = new Image();
            img.onload = () => resolve(img);
            img.onerror = () => reject(new Error(`Failed to load image from URL: ${url}`));
            img.src = url;
        });
    };

    useEffect(() => {
        const loadImages = async () => {
            for (const item of items) {
                if (item.tvgLogo) {
                    try {
                        const img = await loadImage(item.tvgLogo);
                        setImages(prevImages => ({
                            ...prevImages,
                            [item.tvgLogo]: img.src
                        }));
                    } catch (error) {
                        console.log('Failed to load image', error);
                    }
                }
            }
        };

        loadImages();
    }, [items]);

    useEffect(() => {
        let scrollSpeed = 0; // Variable to track scroll speed
        let animationFrameId = null;

        const scrollContent = () => {
            if (scrollSpeed !== 0 && channelsRef.current) {
                console.log(scrollSpeed);
                channelsRef.current.scrollBy({ top: scrollSpeed, behavior: 'auto' });
                console.log(scrollSpeed);

            }
            animationFrameId = requestAnimationFrame(scrollContent); // Loop the scrolling
        };

        // Start the background scrolling task
        scrollContent();

        const handleMouseEnterTop = () => {
            scrollSpeed = -20; // Negative speed for upward scrolling (adjust as needed)
        };

        const handleMouseEnterBottom = () => {
            scrollSpeed = 20; // Positive speed for downward scrolling (adjust as needed)
        };

        const handleMouseLeave = () => {
            scrollSpeed = 0; // Stop scrolling when mouse leaves trigger areas
        };

        // Attach event listeners to the trigger areas
        if (topTriggerRef.current && bottomTriggerRef.current) {
            topTriggerRef.current.addEventListener('mouseenter', handleMouseEnterTop);
            bottomTriggerRef.current.addEventListener('mouseenter', handleMouseEnterBottom);
            topTriggerRef.current.addEventListener('mouseleave', handleMouseLeave);
            bottomTriggerRef.current.addEventListener('mouseleave', handleMouseLeave);
        }

        if (channelsRef.current) {
            channelsRef.current.addEventListener('mousewheel', (event) => {
                scrollSpeed = event.deltaY; // Set scroll speed based on mouse wheel delta
            });
        }

        return () => {
            if (animationFrameId) {
                cancelAnimationFrame(animationFrameId); // Clean up animation frame on unmount
            }
            // Remove event listeners to avoid memory leaks
            if (topTriggerRef.current && bottomTriggerRef.current) {
                topTriggerRef.current.removeEventListener('mouseenter', handleMouseEnterTop);
                bottomTriggerRef.current.removeEventListener('mouseenter', handleMouseEnterBottom);
                topTriggerRef.current.removeEventListener('mouseleave', handleMouseLeave);
                bottomTriggerRef.current.removeEventListener('mouseleave', handleMouseLeave);
            }
        };
    }, []);


    return (
        <div className="playlist">
            <div class="mt-3">
                <input
                    type="text"
                    className="form-control"
                    placeholder="Search Channel"
                    value={searchTerm}
                    onChange={e => setSearchTerm(e.target.value)}
                />
            </div>
            <div ref={topTriggerRef} className="mt-3 scroll scroll_down">
                <span>
                    <i class="bi bi-arrow-up-square-fill"></i>
                </span>
            </div>
            <div className="mt-3 channels" ref={channelsRef}>
                {filteredItems.map((item, index) => (
                    <div
                        key={index}
                        className="channel d-flex flex-column p-3"
                        onClick={() => onChannelClick(item)}
                    >
                        {item.tvgLogo && images[item.tvgLogo] ? (
                            <img src={images[item.tvgLogo]} alt={item.tvgName} className="logo" />
                        ) : (
                            <span className="title">{item.tvgName}</span>
                        )}
                    </div>
                ))}
            </div>
            <div ref={bottomTriggerRef} className="mt-3 scroll scroll_up">
                <span>
                    <i class="bi bi-arrow-down-square-fill"></i>
                </span>
            </div>
        </div>
    );
}

export default Playlist;
