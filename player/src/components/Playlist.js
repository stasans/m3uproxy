import React, { useState } from 'react';

function Playlist({ items, onChannelClick, onUpdatePlaylist }) {
    const [searchTerm, setSearchTerm] = useState('');

    const filteredItems = items.filter(item =>
        item.tvgName.toLowerCase().includes(searchTerm.toLowerCase())
    );

    return (
        <div className="col-md-2">
            <input
                type="text"
                className="form-control mb-3"
                placeholder="Search Channel"
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
            />
            <button className="btn btn-primary" onClick={onUpdatePlaylist}>
                <i className="bi bi-arrow-right-circle-fill"></i>
            </button>
            <div className="mt-3">
                {filteredItems.map((item, index) => (
                    <div
                        key={index}
                        className="channel d-flex flex-column p-3"
                        onClick={() => onChannelClick(item)}
                    >
                        {item.tvgLogo && <img src={item.tvgLogo} alt={item.tvgName} className="tv-logo mb-2" />}
                        <span className="tv-title">{item.tvgName}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}

export default Playlist;
