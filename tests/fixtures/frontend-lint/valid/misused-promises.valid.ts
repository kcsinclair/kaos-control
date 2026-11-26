// Positive case: the async handler is wrapped so the outer callback stays
// synchronous (void-returning), and the inner promise is voided.
async function loadData(): Promise<void> {}

export function registerHandler(onClick: () => void): void {
  onClick()
}

registerHandler(() => {
  void loadData()
})
