import React, { Component, createRef } from 'react';

class Playlist extends Component {
    constructor(props) {
        super(props);
        this.channelNameRef = createRef();
        this.channelNumberRef = createRef();


        this.state = {
            channelNumberVisible: true,
            channelNameVisible: true,
            channelName: "",
            channelNumber: 0,
        };
    }

    showChannelNumber = () => {
        this.setState({ channelNumberVisible: true });
    }

    hideChannelNumber = () => {
        this.setState({ channelNumberVisible: false });
    }

    showChannelName = () => {
        this.setState({ channelNameVisible: true });
    }

    hideChannelName = () => {
        this.setState({ channelNameVisible: false });
    }

    setChannelName = (name) => {
        this.setState({ channelName: name });
    }

    setChannelNumber = (number) => {
        this.setState({ channelNumber: number });
    }

    render() {
        const {
            channelNumberVisible,
            channelNameVisible,
            channelName,
            channelNumber
        } = this.state;

        return (
            <div className="overlay">
                <div ref={this.channelNameRef} className="channel-name" style={{
                    opacity: channelNameVisible ? 1 : 0,
                }} >{channelName}</div>
                <div ref={this.channelNumberRef} className="channel-number" style={{
                    opacity: channelNumberVisible ? 1 : 0,
                }} >{channelNumber}</div>
            </div>
        );
    }
}

export default Playlist;
