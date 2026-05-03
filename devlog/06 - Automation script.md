# 2026. 05. 03. - Automation script


# Considerations

Together with my consultant we've gathered what the "script" should be able to do.
These were:
- It should be parameterizable
- Should automatically start everything outside of the cluster necessary for the measurement.
- Should be able to handle multiple measurements to different clusters at the same time.
- After the measurement it should retrieve the statistics from the prometheus servers in each cluster.

## Solution

With these considerations in mind I firstly set out to create a bash script that could be ran with different flags. However we've quickly came to the conclusion that it'd be best if we could store the parameters used for the tests, so they could be easily replicable, and also documentable.

Due to this, the final decision was to use Golang, as it has many supporting user packages which come really handy for us. These packages are [`styx`](https://github.com/go-pluto/styx) for communicating with the prometheus API and [`viper`](https://github.com/spf13/viper) for easy yaml config storage and handling. Additionally we may use [`cobra`](https://github.com/spf13/cobra) to create a user friendly CLI application.