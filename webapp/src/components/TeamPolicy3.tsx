import React, { useState } from 'react'

// import AdminPanelWithButton from './admin_panel_with_button';

import { Button, Form } from 'react-bootstrap';
import { getTeams as loadTeams, getMyTeams } from 'mattermost-redux/actions/teams';
import { ActionFunc } from 'mattermost-redux/types/actions';
import { TeamsWithCount } from 'mattermost-redux/types/teams';
import { useSelector, useDispatch, useStore, shallowEqual } from 'react-redux';
import { GlobalState } from 'mattermost-redux/types/store';
import { getTeams } from 'mattermost-redux/selectors/entities/teams';
import { Team } from 'mattermost-redux/types/teams';
import { IDMappedObjects } from 'mattermost-redux/types/utilities';

type Props = {
        // id?: string;
        // className?: string;
        // onHeaderClick?: React.EventHandler<React.MouseEvent>;
        // titleId: string;
        // titleDefault: string;
        // subtitleId: string;
        // subtitleDefault: string;
        // subtitleValues?: any;
        // button?: React.ReactNode;
        children?: React.ReactNode;
};
export default React.memo((props: Props) => {

        const teams = [
                {
                        id: "team1",
                        display_name: "team name"
                },
                {
                        id: "team2",
                        display_name: "team name 2"
                },
        ]

        type teams_prop = {
                id: string,
                display_name: string,
        }

        // const [selTeams, setSelTeams] = useState(teams)

        const dispatch = useDispatch()

        dispatch(loadTeams())

        const teamsData = useSelector<GlobalState, IDMappedObjects<Team>>((state) => getTeams(state), (prev, curr) => 
                shallowEqual(Object.keys(prev).sort(),Object.keys(curr).sort())
        )
        // console.log(teamsData)

        const team = (props: teams_prop) => (
                <div
                        className='team'
                        key={props.id}
                        style={{ display: "flex", padding: 10 }}
                >
                        <div style={{ flexGrow: 50, margin: 1 }} >
                                <select name="pets" id="pet-select" className={"form-control"} >
                                 
                                        <option value="">--Please choose an team--</option>
                                        <option value="dog">Dog</option>
                                        <option value="cat">Cat</option>
                                        <option value="hamster">Hamster</option>
                                        <option value="parrot">Parrot</option>
                                        <option value="spider">Spider</option>
                                        <option value="goldfish">Goldfish</option>
                                </select>
                        </div >
                        <div style={{ flexGrow: 0.5, margin: 1 }}>
                                <input type={"number"} name={props.id} className={"form-control"} />
                        </div>

                        <div style={{ flexGrow: 0.5, margin: 1 }}>
                                <Button >
                                        {'Remove'}
                                </Button>
                        </div>
                </div >
        )
        return (
                <div>
                        <div
                                className={'AdminPanel clearfix '}
                        >
                                <div
                                        className='header'
                                // onClick={props.onHeaderClick}
                                >

                                        <div>
                                                <h3>
                                                        {"Team Setting"}
                                                </h3>
                                                <div className='mt-2'>
                                                        {"Set specific team rules."}
                                                </div>
                                        </div>
                                        <div className='button'>
                                                <a
                                                        className={'btn btn-primary'}
                                                // onClick={handleOpen}
                                                >
                                                        {"Add Teams"}
                                                </a>
                                        </div>
                                </div>
                                {teams.map((team_data) => team(
                                        {
                                                id: team_data.id,
                                                display_name: team_data.display_name
                                        },
                                ))}
                        </div>
                </div>
        )

})
