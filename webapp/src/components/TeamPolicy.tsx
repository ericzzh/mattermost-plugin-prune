import React, { useEffect, useState } from "react";

// import AdminPanelWithButton from './admin_panel_with_button';

import { Button, Form } from "react-bootstrap";
import {
    getTeams as loadTeams,
    getMyTeams,
} from "mattermost-redux/actions/teams";
import { ActionFunc } from "mattermost-redux/types/actions";
import { TeamsWithCount } from "mattermost-redux/types/teams";
import { useSelector, useDispatch, useStore, shallowEqual } from "react-redux";
import { GlobalState } from "mattermost-redux/types/store";
import { getTeams } from "mattermost-redux/selectors/entities/teams";
import { Team } from "mattermost-redux/types/teams";
import { IDMappedObjects } from "mattermost-redux/types/utilities";
import PropTypes from "prop-types";

function TeamPolicy(props: any) {
    const [value, setValue] = useState(props.value);
    const [message, setMessage] = useState("this is message");

    const handleSave = async () => {
        console.log("I am in handleSAve.");
        let error = { message: "There is a error" };
        return { error };
    };

    useEffect(() => {
        console.log("logging.........");
        console.log(props.registerSaveAction);
        props.registerSaveAction(handleSave);
        return () => props.unRegisterSaveAction(handleSave);
    }, []);

    const handleChange = (e: any) => {
        setValue(e.target.value);
        props.onChange(props.id, e.target.value);
        props.setSaveNeeded();
    };

    return (
        <div>
            <div>
                <textarea
                    id={"teampolicy"}
                    className={"form-control"}
                    onChange={handleChange}
                    value={value}
                    rows={10}
                />
            </div>
            <div style={style.error}>{message}</div>
        </div>
    );
}

TeamPolicy.propTypes = {
    id: PropTypes.string.isRequired,
    label: PropTypes.string.isRequired,
    helpText: PropTypes.node,
    value: PropTypes.any,
    disabled: PropTypes.bool.isRequired,
    config: PropTypes.object.isRequired,
    license: PropTypes.object.isRequired,
    setByEnv: PropTypes.bool.isRequired,
    onChange: PropTypes.func.isRequired,
    registerSaveAction: PropTypes.func.isRequired,
    setSaveNeeded: PropTypes.func.isRequired,
    unRegisterSaveAction: PropTypes.func.isRequired,
};

const style = {
    error: {
        color: "red",
    },
};

export default React.memo(TeamPolicy);
