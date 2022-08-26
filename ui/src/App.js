import {Component} from "react";
import {Container, Stack, Row} from "react-bootstrap";
import Table from "./Table";

class App extends Component {
  render() {
    return (
        <Container fluid>
          <Stack>
            <Row>
              <Table></Table>
            </Row>
          </Stack>
        </Container>
    )
  }
}
export default App;
