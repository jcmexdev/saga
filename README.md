<pre>

sequenceDiagram
    participant C as Customer
    participant G as API Gateway (Orchestrator)
    participant O as Order Service
    participant I as Inventory Service
    participant P as Payment Service

    C->>G: POST /orders
    rect rgb(240, 240, 240)
        note right of G: Saga Execution
        G->>O: 1. Create Order (PENDING)
        O-->>G: Success (ID)
        G->>I: 2. Reserve Stock
        I-->>G: Success
        G->>P: 3. Process Payment
        P-->>G: Success
        G->>O: 4. Confirm Order (CONFIRMED)
        O-->>G: Success
    end
    G-->>C: 202 Accepted (Order ID)

    </pre>
