import {Component} from "react";
import BootstrapTable from "react-bootstrap-table-next"
import {Proposals} from "./Proposals";
import getUnicodeFlagIcon from 'country-flag-icons/unicode'
import paginationFactory from 'react-bootstrap-table2-paginator';
import filterFactory, {textFilter} from 'react-bootstrap-table2-filter';

class Table extends Component {
    constructor(props) {
        super(props);
        this.state = {
            proposals: []
        }
    }

    componentDidMount() {
        Proposals().then((proposals) => {
            this.setState({proposals: proposals})
        })
    }

    render() {
        return (
            <div>
                <BootstrapTable
                    keyField='provider_id'
                    data={this.state.proposals}
                    columns={proposalColumns}
                    expandRow={expandRow}
                    pagination={ pagination }
                    filter={ filterFactory() }
                    striped
                    hover
                    condensed
                />
            </div>
        )
    }
}

const proposalColumns = [
    {
        dataField: 'provider_id',
        text: "ID",
        filter: textFilter(),
        formatter: function (cell, row) {
            return (
                <div>
                    <a href="#" onClick={(e) => {e.preventDefault(); navigator.clipboard.writeText(row.provider_id)}}>
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"
                             className="bi bi-clipboard-check-fill" viewBox="0 0 16 16">
                            <path
                                d="M6.5 0A1.5 1.5 0 0 0 5 1.5v1A1.5 1.5 0 0 0 6.5 4h3A1.5 1.5 0 0 0 11 2.5v-1A1.5 1.5 0 0 0 9.5 0h-3Zm3 1a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5h-3a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3Z"/>
                            <path
                                d="M4 1.5H3a2 2 0 0 0-2 2V14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V3.5a2 2 0 0 0-2-2h-1v1A2.5 2.5 0 0 1 9.5 5h-3A2.5 2.5 0 0 1 4 2.5v-1Zm6.854 7.354-3 3a.5.5 0 0 1-.708 0l-1.5-1.5a.5.5 0 0 1 .708-.708L7.5 10.793l2.646-2.647a.5.5 0 0 1 .708.708Z"/>
                        </svg>
                    </a>
                    &nbsp;
                    {row.provider_id}
                </div>
            )
        }

    },
    {
        text: "Quality",
        formatter: function (cell, row) {
            var quality = '⬛'
            if (row.quality.quality === 0) {
                quality = '🟥'
            } else if (row.quality.quality < 1) {
                quality = '🟥'
            } else if (row.quality.quality < 2) {
                quality = '🟧'
            } else if (row.quality.quality < 2.5) {
                quality = '🟨'
            } else {
                quality = '🟩'
            }
            return (
                <div>
                    {quality}
                </div>
            )
        },
    },
    {
        text: "Country",
        formatter: function (cell, row) {
            return ( `${getUnicodeFlagIcon(row.location.country)} ${row.location.city} (${row.location.isp})`)
        }
    },
    {
        text: "IP Type",
        formatter: function (cell, row) {
            const type = row.location.ip_type
            if (type === 'residential') {
                return (<span className="badge text-bg-success">Residential</span>)
            } else if (type === 'hosting') {
                return (<span className="badge text-bg-warning">Hosting</span>)
            }

            return (<span className="badge text-bg-dark">{row.location.ip_type}</span>)
        }
    },
    {
        text: "VPN Type",
        formatter: function (cell, row) {
            const type = row.service_type
            if (type === 'wireguard') {
                return (<span className="badge text-bg-primary">Wireguard</span>)
            } else if (type === 'openvpn') {
                return (<span className="badge text-bg-secondary">Openvpn</span>)
            }

            return (<span className="badge text-bg-dark">{row.service_type}</span>)
        }
    },

]

const expandRow = {
    renderer: row => (
        <div>
            <p> Quality: {Number(parseFloat(row.quality.quality).toFixed(2))}/3 </p>
            <p> Latency: {Number(parseFloat(row.quality.latency).toFixed(0))}ms </p>
            <p> Bandwidth: {Number(parseFloat(row.quality.latency).toFixed(2))}Mbps </p>
            <p> Uptime: {Number(parseFloat(row.quality.uptime).toFixed(2))}/24 </p>
        </div>
    ),
        showExpandColumn: true,
        expandByColumnOnly: true
};

const pagination = paginationFactory({
    sizePerPage: 100
});


export default Table