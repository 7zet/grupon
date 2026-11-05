# Grupon handover

Grupon is a polished demonstration project. It has no real users, real money processing, or legal/compliance delivery scope. One owner tests all roles locally: buyer, merchant owner, cashier, and admin.

## Not included (interfaces ready)

- Real PSP integration, payment reconciliation, refunds/chargebacks with external providers, and real-money operating procedures.
- Fiscalization, MXIK/VAT data, and receipt compliance.
- Bank-account change protection and the related delayed-approval flow.
- Legal, regulatory, privacy, merchant-contract, and launch gates.
- A real pilot launch or seven-day staging run.
- Dockerfiles and `docker-compose.prod.yml` are optional and not verified; deployment and CI use native runtimes.

The existing payment, ledger, finance, SMS-sender, and deployment interfaces are intentionally retained so a later production project can implement these capabilities without redesigning the demo’s core flows.
