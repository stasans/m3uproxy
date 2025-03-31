import React, { Component, createRef } from 'react';

class Playlist extends Component {
    constructor(props) {
        super(props);
        this.channelNameRef = createRef();
        this.channelNumberRef = createRef();


        this.state = {
            channelNumberVisible: true,
            channelNameVisible: true,
            currentChannel: null,
        };
    }

    componentDidMount() {
    }

    componentDidUpdate(prevProps, prevState) {
    }

    componentWillUnmount() {
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

    setCurrentChannel = (channel) => {
        this.setState({ currentChannel: channel });
    }

    render() {
        const { currentChannel, channelNumberVisible, channelNameVisible } = this.state;

        return (
            <div className="overlay">
                <div ref={this.channelNameRef} className="channel-name" style={{
                    opacity: channelNameVisible ? 1 : 0,
                    transition: channelNameVisible ? "" : "opacity 2s ease-out"
                }} >{currentChannel ? currentChannel.tvgName : ""}</div>
                <div ref={this.channelNumberRef} className="channel-number" style={{
                    opacity: channelNumberVisible ? 1 : 0,
                    transition: channelNumberVisible ? "" : "opacity 2s ease-out"
                }} >{currentChannel ? currentChannel.channel_num + 1 : 0}</div>
            </div>
        );
    }
}

export default Playlist;
