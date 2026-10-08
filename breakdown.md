Client POST /jobs
Authorization: Bearer <token>
Content-Type: application/json
{type:"cmd type", payload:"cmd payload"}
\/
Server Response: 200 OK
Content-Type: application/json
{job_id:"job_id", status:"status"}
\/
Worker picks up job; executes job; status {"completed","failed","in_progress"}; store in DB for 7 days
\/
Client GET /jobs/:id
Authorization: Bearer <token>
\/
Server Response: 200 OK
Content-Type: application/json
{job_id:"job_id", status:"status", result:"cmd result"}
\/
Client DELETE /jobs/:id
Authorization: Bearer <token>
\/
Server Response: 200 OK