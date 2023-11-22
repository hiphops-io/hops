// Event logs table data
export let tableData = [
    {
        timestamp: '2023-10-10 10:00 AM',
        eventId: 'Id',
        data: 'Data',
        status: 'Status',
        pipelineSteps: [
            {
                name: 'Step 1',
                status: 'Completed',
                executionTime: '2023-10-10 10:01 AM',
                JSON: {
                    users: [
                        {
                            id: 1,
                            name: 'John Doe',
                            email: 'johndoe@example.com',
                            address: {
                                street: '123 Elm St',
                                city: 'Anytown',
                                state: 'CA',
                                zip: '12345'
                            },
                            orders: [
                                {
                                    orderId: 'A123',
                                    product: 'Laptop',
                                    price: 1200.0,
                                    quantity: 1
                                },
                                {
                                    orderId: 'A124',
                                    product: 'Mouse',
                                    price: 20.0,
                                    quantity: 2
                                }
                            ]
                        }
                    ]
                }
            },
            {
                name: 'Step 2',
                status: 'In progress',
                executionTime: '2023-10-10 10:02 AM',
                JSON: 'JSON 2'
            },
            {
                name: 'Step 3',
                status: 'Failed',
                executionTime: '2023-10-10 10:03 AM',
                JSON: 'JSON 3'
            },
            {
                name: 'Step 4',
                status: 'DNR',
                executionTime: '2023-10-10 10:04 AM',
                JSON: 'JSON 4'
            }
        ]
    },
    {
        timestamp: '2023-10-10 10:01 AM',
        eventId: 'Id',
        data: 'Data',
        status: 'Status',
        pipelineSteps: [
            {
                name: 'Step 5',
                status: 'Failed',
                executionTime: '2023-10-10 10:01 AM',
                JSON: 'JSON'
            },
            {
                name: 'Step 6',
                status: 'Completed',
                executionTime: '2023-10-10 10:02 AM',
                JSON: 'JSON'
            },
            {
                name: 'Step 7',
                status: 'In progress',
                executionTime: '2023-10-10 10:03 AM',
                JSON: 'JSON'
            },
            {
                name: 'Step 8',
                status: 'DNR',
                executionTime: '2023-10-10 10:04 AM',
                JSON: 'JSON'
            }
        ]
    }
];