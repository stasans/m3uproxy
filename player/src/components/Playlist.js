import React, { useState, useRef, useEffect } from 'react';

function Playlist({ items, onChannelClick }) {
    const channelsRef = useRef(null);
    const topTriggerRef = useRef(null);
    const bottomTriggerRef = useRef(null);

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
        let scrollDir = 0;
        let scrollSpeed = 0;
        let animationFrameId = null;

        const scrollContent = () => {
            if (!channelsRef.current) {
                animationFrameId = requestAnimationFrame(scrollContent);
                return;
            }
            if ((scrollDir === -1 || scrollDir === 1)) {
                scrollSpeed += scrollDir * 0.5;
                if (Math.abs(scrollSpeed) > 20) {
                    scrollSpeed = 20 * scrollDir;
                }
            } else {
                scrollSpeed *= 0.9; // Decelerate scrolling
            }
            if (scrollSpeed !== 0) {
                channelsRef.current.scrollBy({ top: scrollSpeed, behavior: 'auto' });
            }
            animationFrameId = requestAnimationFrame(scrollContent);
        };

        // Start the background scrolling task
        scrollContent();

        const handleMouseEnterTop = () => {
            scrollDir = -1;
        };

        const handleMouseEnterBottom = () => {
            scrollDir = 1;
        };

        const handleMouseLeave = () => {
            scrollDir = 0;
        };

        // Attach event listeners to the trigger areas when on a desktop browser
        if (!window.globalConfig.isMobile) {
            if (topTriggerRef.current && bottomTriggerRef.current) {
                topTriggerRef.current.addEventListener('mouseenter', handleMouseEnterTop);
                bottomTriggerRef.current.addEventListener('mouseenter', handleMouseEnterBottom);
                topTriggerRef.current.addEventListener('mouseleave', handleMouseLeave);
                bottomTriggerRef.current.addEventListener('mouseleave', handleMouseLeave);
            }

            if (channelsRef.current) {
                if (navigator.userAgent.toLowerCase().indexOf('firefox') > -1) {
                    channelsRef.current.addEventListener('wheel', (event) => {
                        scrollSpeed = event.deltaY;
                    });
                } else {
                    channelsRef.current.addEventListener('mousewheel', (event) => {
                        scrollSpeed = event.deltaY;
                    });
                }
            }
        }

        // If on a mobile device, hide the scroll triggers and enable touch scrolling
        if (window.globalConfig.isMobile) {
            // Hide scroll triggers on mobile devices and tablets
            if (topTriggerRef.current && bottomTriggerRef.current) {
                topTriggerRef.current.style.display = 'none';
                bottomTriggerRef.current.style.display = 'none';
            }
            // on mobile devices, scroll the channels container when the user swipes
            if (channelsRef.current) {
                let touchStartY = 0;
                channelsRef.current.addEventListener('touchstart', (event) => {
                    touchStartY = event.touches[0].clientY;
                });
                channelsRef.current.addEventListener('touchmove', (event) => {
                    const touchEndY = event.touches[0].clientY;
                    const deltaY = touchEndY - touchStartY;
                    channelsRef.current.scrollBy({ top: -deltaY, behavior: 'auto' });
                    touchStartY = touchEndY;
                });
            }
        }

        return () => {
            if (animationFrameId) {
                cancelAnimationFrame(animationFrameId); // Clean up animation frame on unmount
            }
            // Remove event listeners to avoid memory leaks
            if (channelsRef.current) {
                if (topTriggerRef.current && bottomTriggerRef.current) {
                    topTriggerRef.current.removeEventListener('mouseenter', handleMouseEnterTop);
                    bottomTriggerRef.current.removeEventListener('mouseenter', handleMouseEnterBottom);
                    topTriggerRef.current.removeEventListener('mouseleave', handleMouseLeave);
                    bottomTriggerRef.current.removeEventListener('mouseleave', handleMouseLeave);
                }

                if (navigator.userAgent.toLowerCase().indexOf('mobile') === -1) {
                    if (navigator.userAgent.toLowerCase().indexOf('firefox') > -1) {
                        channelsRef.current.removeEventListener('wheel', (event) => {
                            scrollSpeed = event.deltaY;
                        });
                    } else {
                        channelsRef.current.removeEventListener('mousewheel', (event) => {
                            scrollSpeed = event.deltaY;
                        });
                    }
                } else {
                    channelsRef.current.removeEventListener('touchstart', (event) => {
                        touchStartY = event.touches[0].clientY;
                    });
                    channelsRef.current.removeEventListener('touchmove', (event) => {
                        const touchEndY = event.touches[0].clientY;
                        const deltaY = touchEndY - touchStartY;
                        channelsRef.current.scrollBy({ top: -deltaY, behavior: 'auto' });
                        touchStartY = touchEndY;
                    });
                }
            }
        };
    }, []);

    return (
        <div className="playlist">
            <div ref={topTriggerRef} className="scroll scroll_down">
                <span>
                    <i className="bi bi-arrow-up-square-fill"></i>
                </span>
            </div>
            <div className="channels" ref={channelsRef}>
                {items.map((item, index) => (
                    <div
                        key={index}
                        className="channel"
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
            <div ref={bottomTriggerRef} className="scroll scroll_up">
                <span>
                    <i className="bi bi-arrow-down-square-fill"></i>
                </span>
            </div>
        </div>
    );
}

export default Playlist;
