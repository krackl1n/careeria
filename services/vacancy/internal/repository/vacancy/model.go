package vacancy

const columns = `v.id, v.company_id, v.author_id, v.title, v.description, v.work_format,
	v.location, v.required_experience, v.employment_type, v.status, v.version,
	v.created_at, v.updated_at, v.first_published_at, v.published_at, v.closed_at,
	(v.compensation_min * 100)::bigint, (v.compensation_max * 100)::bigint,
	v.compensation_currency, v.compensation_period, v.compensation_tax_basis, v.compensation_bonus,
	COALESCE((SELECT jsonb_agg(jsonb_build_object('key', s.skill_key, 'displayName', s.display_name)
		ORDER BY s.skill_key) FROM public.vacancy_skills s WHERE s.vacancy_id = v.id), '[]'::jsonb)`
