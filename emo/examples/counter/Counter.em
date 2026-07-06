// Counter.em — emo 0.1 SDK counter example
// Edit and save — the dev server pushes the new view tree to your device.

component Counter {
  state count = 0

  render {
    <Column className="container">
      <Text fontSize={28} fontWeight="bold">emo counter</Text>
      <Text fontSize={18}>Count: {count}</Text>
      <Row className="buttonRow">
        <Button onClick={() => count = count - 1}>Decrement</Button>
        <Button onClick={() => count = count + 1}>Increment</Button>
      </Row>
      <Button onClick={() => count = 0}>Reset</Button>
      <Divider />
      <Text className="hint">Edit Counter.em and save for live reload!</Text>
    </Column>
  }
}

style "./Counter.css"
