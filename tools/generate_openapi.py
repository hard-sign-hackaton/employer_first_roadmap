"""Build the checked-in OpenAPI document from the current employer API contract.

Run from the repository root: python tools/generate_openapi.py
"""

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def ref(name):
    return {"$ref": f"#/components/schemas/{name}"}


def obj(properties, required=()):
    result = {"type": "object", "properties": properties}
    if required:
        result["required"] = list(required)
    return result


def arr(item):
    return {"type": "array", "items": item}


I = {"type": "integer", "format": "int64"}
S = {"type": "string"}
B = {"type": "boolean"}
T = {"type": "string", "format": "date-time"}

schemas = {
    "Error": obj({"error": S}, ["error"]),
    "Health": obj({"status": {"type": "string", "enum": ["ok"]}}, ["status"]),
    "Account": obj({"id": I, "login": S, "role": {"type": "string", "enum": ["admin", "employer"]}, "company_id": I, "is_active": B, "token": {**S, "description": "Only returned on account creation or token rotation."}}, ["id", "login", "role", "is_active"]),
    "CreateAccount": obj({"login": S, "role": {"type": "string", "enum": ["admin", "employer"]}, "company_id": I}, ["login", "role"]),
    "UpdateAccount": obj({"login": S, "company_id": I, "is_active": B}),
    "CompanyInput": obj({"name": S, "description": S, "website_url": {"type": "string", "format": "uri"}}, ["name"]),
    "Company": obj({"id": I, "name": S, "description": S, "website_url": S, "is_active": B}, ["id", "name", "description", "website_url", "is_active"]),
    "DirectionInput": obj({"name": S, "description": S}, ["name"]),
    "TagWeight": obj({"interest_tag_id": I, "weight": {"type": "number", "format": "double"}}, ["interest_tag_id", "weight"]),
    "Direction": obj({"id": I, "company_id": I, "name": S, "description": S, "is_active": B, "interest_tags": arr(ref("TagWeight")), "education_program_ids": arr(I)}, ["id", "company_id", "name", "description", "is_active", "interest_tags", "education_program_ids"]),
    "DirectionPrograms": obj({"education_program_ids": arr(I)}, ["education_program_ids"]),
    "DirectionTags": obj({"interest_tags": arr(ref("TagWeight"))}, ["interest_tags"]),
    "OpportunityInput": obj({"career_direction_id": I, "type": {"type": "string", "enum": ["internship", "practice", "project", "hackathon", "targeted_training"]}, "name": S, "description": S, "url": {"type": "string", "format": "uri"}, "active_listing_url": S, "source_checked_at": T, "work_format": {"type": "string", "enum": ["onsite", "hybrid", "remote"]}, "min_study_year": {"type": "integer", "minimum": 1, "maximum": 6}, "region_id": I, "is_active": B}, ["career_direction_id", "type", "name", "description", "url", "work_format", "min_study_year", "is_active"]),
    "Opportunity": {"allOf": [ref("OpportunityInput"), obj({"id": I, "updated_at": T}, ["id", "updated_at"])]},
    "UniversityInput": obj({"region_id": I, "name": S, "description": S, "website_url": S}, ["region_id", "name"]),
    "University": {"allOf": [ref("UniversityInput"), obj({"id": I}, ["id"])]},
    "ProgramInput": obj({"university_id": I, "code": S, "name": S, "description": S}, ["university_id", "name"]),
    "Program": {"allOf": [ref("ProgramInput"), obj({"id": I}, ["id"])]},
    "NamedReference": obj({"id": I, "name": S}, ["id", "name"]),
    "ReferenceData": obj({"regions": arr(ref("NamedReference")), "interest_tags": arr(ref("NamedReference")), "exam_subjects": arr(ref("NamedReference"))}, ["regions", "interest_tags", "exam_subjects"]),
    "ExamItem": obj({"exam_subject_id": I, "min_score": {"type": "integer", "minimum": 0, "maximum": 100}}, ["exam_subject_id"]),
    "ExamCombinationInput": obj({"education_program_id": I, "admission_year": {"type": "integer", "minimum": 2020, "maximum": 2100}, "name": S, "items": arr(ref("ExamItem"))}, ["education_program_id", "admission_year", "items"]),
    "ExamCombination": {"allOf": [ref("ExamCombinationInput"), obj({"id": I}, ["id"])]},
    "AdmissionScoreInput": obj({"education_program_id": I, "admission_year": {"type": "integer", "minimum": 2020, "maximum": 2100}, "budget_passing_score": {"type": "integer", "minimum": 0, "maximum": 400}, "paid_passing_score": {"type": "integer", "minimum": 0, "maximum": 400}}, ["education_program_id", "admission_year"]),
    "AdmissionScore": obj({"education_program_id": I, "admission_year": {"type": "integer", "minimum": 2020, "maximum": 2100}, "budget_passing_score": {"type": "integer", "minimum": 0, "maximum": 400}, "paid_passing_score": {"type": "integer", "minimum": 0, "maximum": 400}, "source_url": S, "source_checked_at": T}, ["education_program_id", "admission_year"]),
    "AdmissionRule": obj({"admission_year": {"type": "integer", "minimum": 2020, "maximum": 2100}, "max_universities": {"type": "integer", "minimum": 1}, "max_programs_per_university": {"type": "integer", "minimum": 1}}, ["admission_year", "max_universities", "max_programs_per_university"]),
    "RoadmapStep": obj({"step_type": S, "title": S, "description": S}, ["step_type", "title"]),
    "RoadmapTemplateInput": obj({"career_direction_id": I, "name": S, "steps": arr(ref("RoadmapStep"))}, ["career_direction_id", "name", "steps"]),
    "RoadmapTemplate": obj({"id": I, "career_direction_id": I, "name": S, "version": {"type": "integer"}, "is_active": B, "steps": arr(ref("RoadmapStep"))}, ["id", "career_direction_id", "name", "version", "is_active"]),
    "ArchiveInput": obj({"is_active": B}, ["is_active"]),
    "FeedbackInput": obj({"status": {"type": "string", "enum": ["under_review", "interview", "accepted", "rejected"]}, "message": S, "contact": S}, ["status", "message"]),
    "Feedback": obj({"id": I, "status": S, "message": S, "contact": S, "created_at": T}, ["id", "status", "created_at"]),
    "Application": obj({"application_id": {**I, "description": "ID of the employer opportunity attempt; use this in PATCH /employer/applications/{applicationID}."}, "roadmap_id": I, "user_id": I, "grade": {"type": "integer"}, "user_region": S, "career_direction": S, "opportunity_id": I, "opportunity_name": S, "submitted_at": T, "status": S, "message": S, "contact": S, "feedback_history": arr(ref("Feedback"))}, ["application_id", "roadmap_id", "user_id", "grade", "user_region", "career_direction", "opportunity_id", "opportunity_name", "submitted_at", "status", "feedback_history"]),
    "Created": obj({"status": {"type": "string", "enum": ["created"]}}, ["status"]),
}


def listing(key, schema):
    return obj({key: arr(ref(schema))}, [key])


schemas.update({
    "Accounts": listing("accounts", "Account"), "Companies": listing("companies", "Company"),
    "Directions": listing("directions", "Direction"), "Opportunities": listing("opportunities", "Opportunity"),
    "Applications": listing("applications", "Application"), "Universities": listing("universities", "University"),
    "Programs": listing("education_programs", "Program"), "ExamCombinations": listing("exam_combinations", "ExamCombination"),
    "AdmissionScores": listing("admission_scores", "AdmissionScore"), "AdmissionRules": listing("admission_rules", "AdmissionRule"),
    "RoadmapTemplates": listing("roadmap_templates", "RoadmapTemplate"),
})

# Illustrative data for Swagger's Example Value. IDs and the token below are
# intentionally fictional: callers must use IDs and credentials from their API.
examples = {
    "Error": {"error": "not found: career direction"},
    "Health": {"status": "ok"},
    "Account": {"id": 42, "login": "north-lab-hr", "role": "employer", "company_id": 101, "is_active": True},
    "CreateAccount": {"login": "north-lab-hr", "role": "employer", "company_id": 101},
    "UpdateAccount": {"login": "north-lab-hr", "company_id": 101, "is_active": True},
    "CompanyInput": {"name": "Лаборатория Север", "description": "Разрабатываем цифровые сервисы для городской среды и приглашаем студентов на практику.", "website_url": "https://example.org/north-lab"},
    "DirectionInput": {"name": "Backend-разработка", "description": "Создание API и сервисов обработки данных на Go."},
    "TagWeight": {"interest_tag_id": 7, "weight": 1.0},
    "DirectionPrograms": {"education_program_ids": [301, 302]},
    "DirectionTags": {"interest_tags": [{"interest_tag_id": 7, "weight": 1.0}, {"interest_tag_id": 9, "weight": 0.7}]},
    "OpportunityInput": {"career_direction_id": 201, "type": "internship", "name": "Стажировка Go-разработчика", "description": "Наставник, небольшие продуктовые задачи и гибкий график для студентов от 2 курса.", "url": "https://example.org/north-lab/careers", "active_listing_url": "https://example.org/north-lab/internship/go", "work_format": "hybrid", "min_study_year": 2, "region_id": 1, "is_active": True},
    "UniversityInput": {"region_id": 1, "name": "Университет цифровых технологий", "description": "Учебный пример вуза для демонстрации API.", "website_url": "https://example.org/university"},
    "ProgramInput": {"university_id": 401, "code": "09.03.01", "name": "Информатика и вычислительная техника", "description": "Алгоритмы, базы данных и разработка программных систем."},
    "NamedReference": {"id": 1, "name": "Москва"},
    "ReferenceData": {"regions": [{"id": 1, "name": "Москва"}], "interest_tags": [{"id": 7, "name": "Программирование"}], "exam_subjects": [{"id": 3, "name": "Информатика"}]},
    "ExamItem": {"exam_subject_id": 3, "min_score": 45},
    "ExamCombinationInput": {"education_program_id": 301, "admission_year": 2027, "name": "Математика, русский язык, информатика", "items": [{"exam_subject_id": 1, "min_score": 40}, {"exam_subject_id": 2, "min_score": 40}, {"exam_subject_id": 3, "min_score": 45}]},
    "AdmissionScoreInput": {"education_program_id": 301, "admission_year": 2026, "budget_passing_score": 267, "paid_passing_score": 211},
    "AdmissionScore": {"education_program_id": 301, "admission_year": 2026, "budget_passing_score": 267, "paid_passing_score": 211, "source_url": "https://example.org/university/admission"},
    "AdmissionRule": {"admission_year": 2027, "max_universities": 5, "max_programs_per_university": 5},
    "RoadmapStep": {"step_type": "prepare_for_exams", "title": "Подготовиться к ЕГЭ", "description": "Составить план подготовки по математике, русскому языку и информатике."},
    "RoadmapTemplateInput": {"career_direction_id": 201, "name": "Путь в backend-разработку", "steps": [{"step_type": "prepare_for_exams", "title": "Подготовиться к ЕГЭ", "description": "Составить план подготовки по профильным предметам."}, {"step_type": "employer_experience", "title": "Пройти стажировку", "description": "Откликнуться на стажировку и выполнить учебный проект."}]},
    "ArchiveInput": {"is_active": False},
    "FeedbackInput": {"status": "interview", "message": "Спасибо за отклик! Приглашаем на интервью во вторник в 15:00.", "contact": "hr@example.org"},
    "Feedback": {"id": 901, "status": "interview", "message": "Спасибо за отклик! Приглашаем на интервью во вторник в 15:00.", "contact": "hr@example.org", "created_at": "2026-09-30T12:00:00Z"},
    "Application": {"application_id": 801, "roadmap_id": 701, "user_id": 601, "grade": 11, "user_region": "Москва", "career_direction": "Backend-разработка", "opportunity_id": 501, "opportunity_name": "Стажировка Go-разработчика", "submitted_at": "2026-09-29T10:30:00Z", "status": "interview", "message": "Спасибо за отклик! Приглашаем на интервью во вторник в 15:00.", "contact": "hr@example.org", "feedback_history": [{"id": 901, "status": "interview", "message": "Спасибо за отклик! Приглашаем на интервью во вторник в 15:00.", "contact": "hr@example.org", "created_at": "2026-09-30T12:00:00Z"}]},
    "Created": {"status": "created"},
}
examples["Company"] = {"id": 101, **examples["CompanyInput"], "is_active": True}
examples["Direction"] = {"id": 201, "company_id": 101, **examples["DirectionInput"], "is_active": True, "interest_tags": examples["DirectionTags"]["interest_tags"], "education_program_ids": [301, 302]}
examples["Opportunity"] = {"id": 501, **examples["OpportunityInput"], "updated_at": "2026-09-30T12:00:00Z"}
examples["University"] = {"id": 401, **examples["UniversityInput"]}
examples["Program"] = {"id": 301, **examples["ProgramInput"]}
examples["ExamCombination"] = {"id": 701, **examples["ExamCombinationInput"]}
examples["RoadmapTemplate"] = {"id": 1001, "career_direction_id": 201, "name": "Путь в backend-разработку", "version": 1, "is_active": True, "steps": examples["RoadmapTemplateInput"]["steps"]}
for collection, key, item in (
    ("Accounts", "accounts", "Account"), ("Companies", "companies", "Company"),
    ("Directions", "directions", "Direction"), ("Opportunities", "opportunities", "Opportunity"),
    ("Applications", "applications", "Application"), ("Universities", "universities", "University"),
    ("Programs", "education_programs", "Program"), ("ExamCombinations", "exam_combinations", "ExamCombination"),
    ("AdmissionScores", "admission_scores", "AdmissionScore"), ("AdmissionRules", "admission_rules", "AdmissionRule"),
    ("RoadmapTemplates", "roadmap_templates", "RoadmapTemplate"),
):
    examples[collection] = {key: [examples[item]]}
for name, example in examples.items():
    # Swagger UI still renders Schema Object `example` more consistently than
    # JSON Schema's `examples` array; publish both for OpenAPI 3.1 viewers.
    schemas[name]["example"] = example
    schemas[name]["examples"] = [example]

# method, path, role, request schema, success schema, status, purpose
routes = [
    ("GET", "/healthz", "public", None, "Health", 200, "Process health"),
    ("GET", "/api/v1/me", "authenticated", None, "Account", 200, "Current API account"),
    ("GET", "/api/v1/reference-data", "authenticated", None, "ReferenceData", 200, "Regions, interests and exam subjects"),
    ("GET", "/api/v1/education-programs", "authenticated", None, "Programs", 200, "Education programs"),
    ("GET", "/api/v1/employer/company", "employer", None, "Company", 200, "Own company"),
    ("PUT", "/api/v1/employer/company", "employer", "CompanyInput", "Company", 200, "Update own company"),
    ("GET", "/api/v1/employer/directions", "employer", None, "Directions", 200, "Own career directions"),
    ("POST", "/api/v1/employer/directions", "employer", "DirectionInput", "Direction", 201, "Create direction"),
    ("PATCH", "/api/v1/employer/directions/{directionID}", "employer", "DirectionInput", "Direction", 200, "Update direction"),
    ("PUT", "/api/v1/employer/directions/{directionID}/education-programs", "employer", "DirectionPrograms", None, 204, "Replace direction programs"),
    ("PUT", "/api/v1/employer/directions/{directionID}/interest-tags", "employer", "DirectionTags", None, 204, "Replace direction interests"),
    ("GET", "/api/v1/employer/opportunities", "employer", None, "Opportunities", 200, "Own opportunities"),
    ("POST", "/api/v1/employer/opportunities", "employer", "OpportunityInput", "Opportunity", 201, "Create opportunity"),
    ("PATCH", "/api/v1/employer/opportunities/{opportunityID}", "employer", "OpportunityInput", "Opportunity", 200, "Update opportunity"),
    ("GET", "/api/v1/employer/applications", "employer", None, "Applications", 200, "Own company applications"),
    ("PATCH", "/api/v1/employer/applications/{applicationID}", "employer", "FeedbackInput", "Feedback", 200, "Append applicant feedback"),
    ("GET", "/api/v1/admin/accounts", "admin", None, "Accounts", 200, "List API accounts"),
    ("POST", "/api/v1/admin/accounts", "admin", "CreateAccount", "Account", 201, "Create API account"),
    ("PATCH", "/api/v1/admin/accounts/{accountID}", "admin", "UpdateAccount", "Account", 200, "Update API account"),
    ("POST", "/api/v1/admin/accounts/{accountID}/rotate-token", "admin", None, "Account", 200, "Rotate API token"),
    ("GET", "/api/v1/admin/companies", "admin", None, "Companies", 200, "List companies"),
    ("POST", "/api/v1/admin/companies", "admin", "CompanyInput", "Company", 201, "Create company"),
    ("PUT", "/api/v1/admin/companies/{companyID}", "admin", "CompanyInput", "Company", 200, "Update company"),
    ("PATCH", "/api/v1/admin/companies/{companyID}/archive", "admin", "ArchiveInput", None, 204, "Activate or archive company"),
    ("POST", "/api/v1/admin/companies/{companyID}/directions", "admin", "DirectionInput", "Direction", 201, "Create company direction"),
    ("PATCH", "/api/v1/admin/companies/{companyID}/directions/{directionID}", "admin", "DirectionInput", "Direction", 200, "Update company direction"),
    ("PATCH", "/api/v1/admin/companies/{companyID}/directions/{directionID}/archive", "admin", "ArchiveInput", None, 204, "Activate or archive direction"),
    ("PUT", "/api/v1/admin/companies/{companyID}/directions/{directionID}/education-programs", "admin", "DirectionPrograms", None, 204, "Replace company direction programs"),
    ("PUT", "/api/v1/admin/companies/{companyID}/directions/{directionID}/interest-tags", "admin", "DirectionTags", None, 204, "Replace company direction interests"),
    ("POST", "/api/v1/admin/companies/{companyID}/opportunities", "admin", "OpportunityInput", "Opportunity", 201, "Create company opportunity"),
    ("PATCH", "/api/v1/admin/companies/{companyID}/opportunities/{opportunityID}", "admin", "OpportunityInput", "Opportunity", 200, "Update company opportunity"),
    ("PATCH", "/api/v1/admin/companies/{companyID}/opportunities/{opportunityID}/archive", "admin", "ArchiveInput", None, 204, "Activate or archive opportunity"),
    ("GET", "/api/v1/admin/universities", "admin", None, "Universities", 200, "List universities"),
    ("POST", "/api/v1/admin/universities", "admin", "UniversityInput", "University", 201, "Create university"),
    ("PUT", "/api/v1/admin/universities/{universityID}", "admin", "UniversityInput", "University", 200, "Update university"),
    ("PATCH", "/api/v1/admin/universities/{universityID}/archive", "admin", "ArchiveInput", None, 204, "Activate or archive university"),
    ("GET", "/api/v1/admin/education-programs", "admin", None, "Programs", 200, "List education programs"),
    ("POST", "/api/v1/admin/education-programs", "admin", "ProgramInput", "Program", 201, "Create education program"),
    ("PUT", "/api/v1/admin/education-programs/{programID}", "admin", "ProgramInput", "Program", 200, "Update education program"),
    ("PATCH", "/api/v1/admin/education-programs/{programID}/archive", "admin", "ArchiveInput", None, 204, "Activate or archive program"),
    ("GET", "/api/v1/admin/admission-scores", "admin", None, "AdmissionScores", 200, "List admission score history"),
    ("PUT", "/api/v1/admin/admission-scores", "admin", "AdmissionScoreInput", None, 204, "Upsert admission score"),
    ("GET", "/api/v1/admin/admission-rules", "admin", None, "AdmissionRules", 200, "List admission rules"),
    ("PUT", "/api/v1/admin/admission-rules", "admin", "AdmissionRule", None, 204, "Upsert admission rule"),
    ("GET", "/api/v1/admin/exam-combinations", "admin", None, "ExamCombinations", 200, "List exam combinations"),
    ("POST", "/api/v1/admin/exam-combinations", "admin", "ExamCombinationInput", "ExamCombination", 201, "Create exam combination"),
    ("PUT", "/api/v1/admin/exam-combinations/{combinationID}", "admin", "ExamCombinationInput", "ExamCombination", 200, "Update exam combination"),
    ("GET", "/api/v1/admin/roadmap-templates", "admin", None, "RoadmapTemplates", 200, "List roadmap templates"),
    ("POST", "/api/v1/admin/roadmap-templates", "admin", "RoadmapTemplateInput", "Created", 201, "Create roadmap template"),
]


def build():
    paths = {}
    for method, path, role, request, response, status, summary in routes:
        operation = {"summary": summary, "operationId": re.sub(r"[^a-zA-Z0-9]+", "_", method.lower() + "_" + path).strip("_"), "tags": [path.split("/")[3] if path.startswith("/api/v1/") and len(path.split("/")) > 3 else "common"], "x-required-role": role}
        if role != "public":
            operation["security"] = [{"bearerAuth": []}]
        if "{" in path:
            operation["parameters"] = [{"name": name, "in": "path", "required": True, "schema": {"type": "integer", "format": "int64", "minimum": 1}} for name in re.findall(r"\{([^}]+)\}", path)]
        if request:
            operation["requestBody"] = {"required": True, "content": {"application/json": {"schema": ref(request), "example": examples[request]}}}
        success = {"description": "Success"}
        if response:
            response_example = examples[response]
            if response == "Account" and (method, path) in (
                ("POST", "/api/v1/admin/accounts"),
                ("POST", "/api/v1/admin/accounts/{accountID}/rotate-token"),
            ):
                response_example = {**response_example, "token": "example-token-not-valid"}
            success["content"] = {"application/json": {"schema": ref(response), "example": response_example}}
        operation["responses"] = {str(status): success}
        for code in ([400] if request or "{" in path else []) + ([401, 403] if role != "public" else []) + [404, 409, 500]:
            operation["responses"][str(code)] = {"description": {400: "Malformed JSON or path parameter", 401: "Missing or invalid token", 403: "Wrong role", 404: "Not found", 409: "Conflict", 500: "Server error"}[code], "content": {"application/json": {"schema": ref("Error")}}}
        paths.setdefault(path, {})[method.lower()] = operation
    return {"openapi": "3.1.0", "info": {"title": "Employer First Roadmap API", "version": "1.0.0", "description": "MAX roadmap bot employer and administrator API. All authenticated endpoints accept Authorization: Bearer <token>."}, "servers": [{"url": "https://135.106.216.99", "description": "Public API"}, {"url": "http://localhost:8080", "description": "Local Docker deployment"}], "paths": paths, "components": {"securitySchemes": {"bearerAuth": {"type": "http", "scheme": "bearer"}}, "schemas": schemas}}


def check_route_coverage():
    source = (ROOT / "employerapi" / "handler.go").read_text(encoding="utf-8")
    registered = set(re.findall(r'm\.Handle(?:Func)?\("(GET|POST|PUT|PATCH|DELETE) ([^\"]+)"', source))
    documented = {(method, path) for method, path, *_ in routes}
    if registered != documented:
        raise SystemExit(f"Route mismatch: missing={registered - documented}; extra={documented - registered}")


def check_examples():
    def fields(schema):
        properties = set(schema.get("properties", {}))
        required = set(schema.get("required", []))
        for part in schema.get("allOf", []):
            if "$ref" in part:
                part = schemas[part["$ref"].split("/")[-1]]
            extra_properties, extra_required = fields(part)
            properties |= extra_properties
            required |= extra_required
        return properties, required

    for name, example in examples.items():
        properties, required = fields(schemas[name])
        if not required <= example.keys() or not example.keys() <= properties:
            raise SystemExit(f"Invalid example for {name}: missing={required - example.keys()}, extra={example.keys() - properties}")
    for method, path, _, request, response, _, _ in routes:
        if request and request not in examples:
            raise SystemExit(f"Missing request example: {method} {path}")
        if response and response not in examples:
            raise SystemExit(f"Missing response example: {method} {path}")


if __name__ == "__main__":
    check_route_coverage()
    check_examples()
    output = ROOT / "openapi.json"
    output.write_text(json.dumps(build(), indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"Wrote {output} ({len(routes)} operations)")
