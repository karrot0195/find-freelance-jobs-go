# Provider Specification Template

## General Information
- **Provider Name**: [e.g., VietnamWorks, TopCV, Upwork]
- **Base URL**: [e.g., https://www.example.com/jobs]
- **Requires Authentication**: [Yes / No]
- **Login URL (if applicable)**: [e.g., https://www.example.com/login]

---

## Selectors & DOM Structure

### 1. Job List Container
- **List Container Selector**: [e.g., `.job-list-item`, `div.job-card`]

### 2. Fields Extraction (Inside Job Container)
1. **Title & URL**:
   - **Selector**: [e.g., `h2.title a`]
   - **Attributes**: 
     - Text content -> `Job Title`
     - `href` attribute -> `Job URL`

2. **Category / Industry**:
   - **Selector**: [e.g., `.category-name`, `span.tag`]
   - **Attributes**: 
     - Text content -> `Job Category`

3. **Price / Budget / Salary**:
   - **Selector**: [e.g., `.salary`, `p.price`]
   - **Attributes**: 
     - Text content -> `Budget / Salary`

4. **Additional Metadata (Optional)**:
   - **Selector**: [e.g., `.location`, `.posted-date`]
   - **Attributes**: 
     - Text content -> `Location / Date`

---

## Pagination Configuration
- **Next Page Selector**: [e.g., `.pagination .next-page`, `ul.pagination li:last-child a`]
- **Pagination Type**: [e.g., Link `href` attribute OR URL parameter pattern like `?page=2`]