import React, { createContext, useContext, useState } from 'react';

const GlobalConfigContext = createContext();

let defaultConfig = {
    logLevel: 'info', // Default log level
}


export const GlobalConfigProvider = ({ children }) => {
    const [globalConfig, setGlobalConfig] = useState(defaultConfig);

    // Function to update the global config
    const updateGlobalConfig = (newConfig) => {
        setGlobalConfig((prevConfig) => ({
            ...prevConfig,
            ...newConfig,
        }));
    }

    return (
        <GlobalConfigContext.Provider value={{ globalConfig, updateGlobalConfig }}>
            {children}
        </GlobalConfigContext.Provider>
    );
};


export const useGlobalConfig = () => useContext(GlobalConfigContext);