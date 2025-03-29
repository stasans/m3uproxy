const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const TerserPlugin = require("terser-webpack-plugin");
const { env } = require('process');
const webpack = require('webpack');
const express = require("express");

module.exports = {
    mode: env.NODE_ENV === 'development' ? 'development' : 'production',
    entry: './src/index.js',
    output: {
        path: path.resolve(__dirname, 'dist'),
        filename: env.NODE_ENV === 'development' ? 'bundle.js' : '[name].bundle.js',
        clean: env.NODE_ENV === 'development' ? false : true,
    },
    optimization: env.NODE_ENV === 'development' ? {} : {
        splitChunks: {
            chunks: 'all',
        },
        minimize: true,
        minimizer: [new TerserPlugin()],
    },
    performance: env.NODE_ENV === 'development' ? {} : {
        hints: false,
        maxEntrypointSize: 512000,
        maxAssetSize: 512000
    },
    module: {
        rules: [
            {
                test: /\.js$/,
                exclude: /node_modules/,
                use: {
                    loader: 'babel-loader',
                },
            },
            {
                test: /\.css$/i,
                use: ['style-loader', 'css-loader'],
            },
        ],
    },
    plugins: [
        new HtmlWebpackPlugin({
            template: './public/index.html',
        }),
        new webpack.DefinePlugin({
            __DEV__: env.NODE_ENV === 'development'
        }),
    ],
    bail: true, // Stops Webpack on the first error
    devServer: {
        static: {
            directory: path.join(__dirname, 'public'), // Adjust this to point to your static files folder
        },
        compress: true,
        port: 3000,
        setupMiddlewares: (middlewares, devServer) => {
            if (!devServer) return middlewares;
            const app = devServer.app;

            app.get("/log", (req, res) => {
                console.log("TV LOG:", req.query.msg);
                res.send("Logged");
            });

            return middlewares;
        },
    },
};
