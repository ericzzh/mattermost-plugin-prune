import React, { useState } from 'react'

// import AdminPanelWithButton from './admin_panel_with_button';

import { Modal } from 'react-bootstrap';
import TeamIcon from 'components/widgets/team_icon/team_icon';

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
export default (props: Props) => {

    let [show, setShow] = useState(false)
    const handleHide = () => setShow(false)
    const handleOpen = () => setShow(true)

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
        isDisabled: boolean,
    }

    const modal = show &&
        <Modal
            show={show}
            onHide={handleHide}
            // onExited={this.handleExit}
            role='dialog'
            aria-labelledby='teamSelectorModalLabel'
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>
                    {"select a team"}
                </Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {"this is body"}
            </Modal.Body>
        </Modal>

    const team = (props: teams_prop) => (
        <div
            className='team'
            key={props.id}

            style={{ display: "flex", padding: 10 }}
        >
            <div className='team-info-block'>
                <div className='team-data'>
                    <div className='title'>{props.display_name}</div>
                </div>
            </div>
            <div style={{ marginRight: 10 }}>
                <input type={"number"} name={props.id} className={"form-control"} />
            </div>
            <a
                className={props.isDisabled ? 'remove disabled' : 'remove'}
            // onClick={handleOpen}
            >
                {'Remove'}
            </a>
        </div>
    )
    return (
        <div>
            {modal}
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
                            onClick={handleOpen}
                        >
                            {"Add Teams"}
                        </a>
                    </div>
                </div>
                {teams.map((team_data) => team(
                    {
                        id: team_data.id,
                        display_name: team_data.display_name,
                        isDisabled: false
                    }
                ))}
            </div>
        </div>
    )

}
