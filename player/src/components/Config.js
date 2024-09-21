import React, { useState, useEffect } from 'react';
import { Modal, Button, Form } from 'react-bootstrap';

const Config = ({ show, onClose, onSave }) => {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');

    useEffect(() => {
        const savedUsername = localStorage.getItem('username');
        const savedPassword = localStorage.getItem('password');
        if (savedUsername) setUsername(savedUsername);
        if (savedPassword) setPassword(savedPassword);
    }, []);

    const handleSave = () => {
        if (username && password) {
            localStorage.setItem('username', username);
            localStorage.setItem('password', password);
            onSave();
            onClose();
        } else {
            alert('Please enter both username and password.');
        }
    };


    return (
        <Modal show={show} onClose={onClose}>
            <Modal.Header closeButton onHide={onClose}>
                <Modal.Title>Configuration</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                <Form>
                    <Form.Group controlId="username">
                        <Form.Label>Username</Form.Label>
                        <Form.Control
                            type="text"
                            placeholder="Enter username"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                        />
                    </Form.Group>
                    <Form.Group controlId="password">
                        <Form.Label>Password</Form.Label>
                        <Form.Control
                            type="password"
                            placeholder="Enter password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                        />
                    </Form.Group>
                </Form>
            </Modal.Body>
            <Modal.Footer>
                <Button variant="secondary" onClick={onClose}>
                    Close
                </Button>
                <Button variant="primary" onClick={handleSave}>
                    Save
                </Button>
            </Modal.Footer>
        </Modal>
    );
};

export default Config;
