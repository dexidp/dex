var path = require('path');
var webpack = require('webpack');

var ExtractTextPlugin = require('extract-text-webpack-plugin');
var OptimizeCssAssetsPlugin = require('optimize-css-assets-webpack-plugin');
var HtmlWebpackPlugin = require('html-webpack-plugin');
module.exports = {
    entry: "./index.js",
    output: {
        path: path.resolve(__dirname, 'dist'),
        filename: "bundle-[hash].js"
    },
    devServer: {
        contentBase: path.join(__dirname, "dist"),
        compress: true,
        port: 9000
    },
    module: {
        loaders: [
            // Include all CSS
            {
                test: /\.css$/,
                use: ExtractTextPlugin.extract({
                    use: 'css-loader'
                })
            },
            // Reference images and fonts by hash to enable long term caching
            {
                test: /\.(woff|woff2|ttf|eot|svg|gif|png|jpg)(\?v=[0-9]\.[0-9]\.[0-9])?$/,
                loader: "file-loader?&name=media/[hash].[ext]"
            }
        ]
    },
    plugins: [
        new webpack.optimize.UglifyJsPlugin({
            minimize: true
        }),
        new webpack.ProvidePlugin({
            $: 'jquery',
            jQuery: 'jquery',
        }),
        new HtmlWebpackPlugin({
            template: 'login.html',
            inject: 'body',
            filename: 'login.html',
            minify: {
                collapseWhitespace: true,
                removeComments: true
            }
        }),
        new ExtractTextPlugin("styles-[hash].css"),
        new OptimizeCssAssetsPlugin({
            cssProcessorOptions: {
                discardComments: {
                    removeAll: true
                }
            }
        })
    ]
};
