/*global React, $*/
"use strict";

// some reactjs code here
var DockerBuild, DockerBuildList;

DockerBuild = React.createClass({
    componentDidMount: function () {
        this.loadBuilds();
        setInterval(this.loadBuilds, this.props.pollInterval);
    },
    render: function () {
        return (
            <div></div>
        );
    }
});

DockerBuildList = React.createClass({
    render: function () {
        var nodes;
        nodes = this.props.builds.map(function (build) {
            return (
                <div class="row">
                    <DockerBuild pollInterval={2000} repo={build.repo} status={build.status} lastBuildError={build.lastBuildError}>
                    </DockerBuild>
                </div>
            );
        });
        return (
            <div class="container">
            </div>
        );
    } 
});
