// AUTO-GENERATED from api/openapi.yaml — DO NOT EDIT BY HAND.
// Regenerate with `npm run gen:api`; the web-ci drift check fails if this is stale.
// (SPEC-200 FR-2002 — the API contract is generated, never hand-written.)

export interface paths {
    "/healthz": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Liveness probe
         * @description Returns 200 as long as the process is serving. Always public.
         */
        get: operations["getHealthz"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/readyz": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Readiness probe
         * @description 200 when the database is reachable, 503 otherwise. Always public.
         */
        get: operations["getReadyz"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/version": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Build metadata */
        get: operations["getVersion"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/register": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Create an account */
        post: operations["registerUser"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/login": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Start a session
         * @description Verifies credentials and sets an HttpOnly session cookie.
         */
        post: operations["loginUser"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/logout": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /** Revoke the current session */
        post: operations["logoutUser"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/auth/me": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** The authenticated caller's identity */
        get: operations["getCurrentUser"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/profile": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Get the caller's investor profile */
        get: operations["getProfile"];
        /** Create or replace the caller's investor profile */
        put: operations["setProfile"];
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/holdings/fii": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List the caller's FII holdings */
        get: operations["listFIIHoldings"];
        put?: never;
        /** Create an FII holding */
        post: operations["createFIIHolding"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/holdings/fii/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /** Update an owned FII holding */
        put: operations["updateFIIHolding"];
        post?: never;
        /** Delete an owned FII holding */
        delete: operations["deleteFIIHolding"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/holdings/fixed-income": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List the caller's fixed-income holdings */
        get: operations["listFixedIncomeHoldings"];
        put?: never;
        /** Create a fixed-income holding */
        post: operations["createFixedIncomeHolding"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/holdings/fixed-income/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        /** Update an owned fixed-income holding */
        put: operations["updateFixedIncomeHolding"];
        post?: never;
        /** Delete an owned fixed-income holding */
        delete: operations["deleteFixedIncomeHolding"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/dashboard": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Compute the caller's portfolio dashboard (summary, allocation, sector exposure)
         * @description Read-only computed view over the caller's holdings and the latest market data.
         *     Money is integer centavos and shares integer basis points (never a float); a held
         *     FII with no current quote is valued at cost basis and listed in `stale_tickers`.
         */
        get: operations["getDashboard"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/insights": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Generate the caller's explainable portfolio insights
         * @description AI insights across categories (portfolio, allocation, market_context), grounded in
         *     computed facts and emitted only through the gated Insighter — every insight carries an
         *     explanation (FR-013) and never a transaction order (FR-014); the response carries the
         *     non-advice disclaimer. `available` is false when the LLM was fully unavailable; an empty
         *     portfolio returns `available:true` with no insights.
         */
        get: operations["getInsights"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/rebalancing": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Generate explainable contribution guidance (areas, computed split, candidates)
         * @description Given a contribution amount, suggest where to focus the new money: areas (each with a
         *     deterministically computed `suggested_share_bps`, summing to 10000), with grounded named
         *     FII candidates nested inside the FII area. The percentages are computed, not generated; the
         *     gated Insighter only explains them — every area/candidate carries an explanation (FR-013)
         *     and never a transaction order (FR-014); the response carries the non-advice disclaimer.
         *     `available` is false on a full LLM outage. `include_asset_shares` opts into an illustrative
         *     per-candidate share. Money is integer centavos; never a float.
         */
        post: operations["postRebalancing"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/health-score": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Compute the caller's reproducible Portfolio Health Score
         * @description A 0–100 health score with a per-factor breakdown (diversification, concentration, liquidity,
         *     goal_alignment, risk_exposure). The score and breakdown are COMPUTED, not LLM-generated, and
         *     reproducible — same inputs (portfolio, profile, macro) yield the same score and identical
         *     breakdown; the score is market-aware (macro is an input). An optional gated AI narrative
         *     explains the result using the live market and never changes the number — `narrative_available`
         *     is false on an LLM outage. Scores and weights are integers; never a float.
         */
        get: operations["getHealthScore"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/projections": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Compute the caller's income & net-worth projections
         * @description Two deterministic, reproducible projections over the current portfolio: a passive-income
         *     projection (monthly/annual, three scenarios) and a net-worth projection (value over the
         *     horizon from current value + reinvested income + the monthly contribution, three scenarios,
         *     as yearly points for charting). Figures are computed, not LLM-generated; each scenario shows
         *     its assumptions; the response is a labelled estimate, never a transaction order. Money is
         *     integer centavos; never a float.
         */
        get: operations["getProjections"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/chat/messages": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Send a conversational-copilot turn (create or continue a thread)
         * @description A multi-turn, fact-grounded chat turn. Omit `thread_id` to start a new thread, or supply one
         *     to continue it. Every reply is grounded in the user's computed facts and emitted only through
         *     the gated Insighter — it carries an `explanation` and the non-advice `disclaimer`, never a
         *     transaction order (FR-013/FR-014). `available` is false when the copilot was temporarily
         *     unavailable (the turn is not persisted; the thread stays readable). Identity is from the
         *     session; the content is length-bounded.
         */
        post: operations["postChatMessage"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/chat/threads": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** List the caller's conversation threads */
        get: operations["listChatThreads"];
        put?: never;
        post?: never;
        /** Clear all the caller's conversation history */
        delete: operations["clearChatThreads"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/chat/threads/{id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /** Read a thread and its messages */
        get: operations["getChatThread"];
        put?: never;
        post?: never;
        /** Delete a thread and its messages */
        delete: operations["deleteChatThread"];
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/market/indicators": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * The latest SELIC/CDI/IPCA reference rates
         * @description Global reference data (no user_id) reused to resolve a fixed-income holding's
         *     effective annual rate (SPEC-109). An indicator with no ingested value yet is
         *     omitted from the response rather than failing the request.
         */
        get: operations["listMarketIndicators"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        /** @description The generic error envelope used by all 4xx/5xx responses. */
        Error: {
            error: string;
        };
        StatusResponse: {
            status: string;
        };
        ReadinessResponse: {
            /**
             * @example ready
             * @example not_ready
             */
            status: string;
            /** @description Per-dependency status (e.g. `{"db":"up"}`). */
            checks: {
                [key: string]: string;
            };
        };
        VersionResponse: {
            version: string;
            commit: string;
            built_at: string;
        };
        Credentials: {
            /** Format: email */
            email: string;
            /** Format: password */
            password: string;
        };
        User: {
            /** Format: uuid */
            id: string;
            /** Format: email */
            email: string;
        };
        /** @description There is deliberately no `user_id` field — identity comes from the session. */
        ProfileRequest: {
            /** @enum {string} */
            risk_profile: "conservative" | "moderate" | "aggressive";
            objectives: ("retirement" | "passive_income" | "wealth_preservation" | "long_term_growth")[];
            horizon_years: number;
        };
        ProfileResponse: {
            /** @enum {string} */
            risk_profile: "conservative" | "moderate" | "aggressive";
            objectives: ("retirement" | "passive_income" | "wealth_preservation" | "long_term_growth")[];
            horizon_years: number;
            /** Format: date-time */
            created_at: string;
            /** Format: date-time */
            updated_at: string;
        };
        FIIHoldingRequest: {
            /**
             * @description The FII ticker (e.g. `HGLG11`).
             * @example HGLG11
             */
            ticker: string;
            /** @description Number of cotas (positive whole number). */
            quantity: number;
            /**
             * Format: int64
             * @description Average cost per cota, in centavos (integer, never a float).
             */
            average_price_centavos: number;
        };
        FIIHoldingResponse: {
            /** Format: uuid */
            id: string;
            ticker: string;
            quantity: number;
            /** Format: int64 */
            average_price_centavos: number;
            /** Format: date-time */
            created_at: string;
            /** Format: date-time */
            updated_at: string;
        };
        ChatMessage: {
            id: string;
            /** @enum {string} */
            role: "user" | "assistant";
            content: string;
            /** @description Present on assistant messages — the explainability gate guarantees it (FR-013). */
            explanation?: string;
            /** Format: date-time */
            created_at: string;
        };
        ChatThreadResponse: {
            id: string;
            title: string;
            /** Format: date-time */
            created_at: string;
            /** Format: date-time */
            updated_at: string;
        };
        ChatThreadDetailResponse: {
            thread: components["schemas"]["ChatThreadResponse"];
            messages: components["schemas"]["ChatMessage"][];
        };
        /** @description The gated assistant reply to a chat turn. */
        ChatReplyResponse: {
            thread_id: string;
            message: components["schemas"]["ChatMessage"];
            /** @description The non-advice disclaimer (FR-014). */
            disclaimer: string;
            /** @description False when the copilot was temporarily unavailable (turn not persisted). */
            available: boolean;
        };
        /** @description AI insights aggregated across categories, with the non-advice disclaimer. */
        InsightsResponse: {
            insights: {
                /** @enum {string} */
                category: "portfolio" | "allocation" | "market_context";
                title: string;
                detail: string;
                /** @description Always present — the explainability gate (FR-013) guarantees it. */
                explanation: string;
            }[];
            /** @description The non-advice disclaimer (FR-014). */
            disclaimer: string;
            /** @description False when the LLM was fully unavailable (insights empty). */
            available: boolean;
        };
        /** @description Deterministic passive-income and net-worth projections across three scenarios. */
        ProjectionsResponse: {
            income: {
                /** @enum {string} */
                scenario: "pessimistic" | "base" | "optimistic";
                /** Format: int64 */
                monthly_centavos: number;
                /** Format: int64 */
                annual_centavos: number;
                assumptions: {
                    yield_adj_bps: number;
                    note: string;
                };
            }[];
            net_worth: {
                /** @enum {string} */
                scenario: "pessimistic" | "base" | "optimistic";
                /** @description Yearly points (year 0 = current value) for charting. */
                points: {
                    year: number;
                    /** Format: int64 */
                    value_centavos: number;
                }[];
                assumptions: {
                    yield_adj_bps: number;
                    /** Format: int64 */
                    monthly_contribution_centavos: number;
                    horizon_years: number;
                    note: string;
                };
            }[];
            /** @description Labels the projection a non-guaranteed estimate (FR-014). */
            disclaimer: string;
        };
        /** @description A reproducible 0–100 Portfolio Health Score with a per-factor breakdown. */
        HealthScoreResponse: {
            /** @description 0–100, computed (never LLM-generated), reproducible. */
            score: number;
            factors: {
                /** @enum {string} */
                name: "diversification" | "concentration" | "liquidity" | "goal_alignment" | "risk_exposure";
                /** @description The factor sub-score, 0–100. */
                score: number;
                /** @description The factor weight, basis points (factors sum to 10000). */
                weight_bps: number;
                /** @description Computed, reproducible — identical for identical inputs (FR-1062). */
                explanation: string;
            }[];
            /** @description Optional gated AI narrative; empty when unavailable. */
            narrative: string;
            /** @description False on a full LLM outage (the score + breakdown are always present). */
            narrative_available: boolean;
            /** @description The non-advice disclaimer accompanying the narrative (FR-014). */
            disclaimer: string;
        };
        /** @description Contribution guidance — areas with a computed split + grounded candidates. */
        RebalancingResponse: {
            areas: {
                /** @enum {string} */
                class: "fii" | "fixed_income";
                /** @description Computed share of the contribution, basis points (areas sum to 10000). */
                suggested_share_bps: number;
                /** Format: int64 */
                suggested_amount_centavos: number;
                title: string;
                detail: string;
                /** @description Always present — the explainability gate (FR-013) guarantees it. */
                explanation: string;
                /** @description Grounded named candidates (FII area only); each a consideration, never an order. */
                candidates: {
                    ticker: string;
                    sector: string;
                    title: string;
                    detail: string;
                    explanation: string;
                    /** @description Present only when include_asset_shares was requested. */
                    illustrative_share_bps?: number;
                }[];
            }[];
            /** @description The non-advice disclaimer (FR-014). */
            disclaimer: string;
            /** @description False when the LLM was fully unavailable (areas empty). */
            available: boolean;
        };
        /** @description Computed portfolio dashboard. All money is integer centavos; all shares are basis points. */
        DashboardResponse: {
            summary: {
                /** Format: int64 */
                total_invested_centavos: number;
                /**
                 * Format: int64
                 * @description The full patrimony / net worth — the sum of every holding's current value.
                 */
                current_value_centavos: number;
                /** Format: int64 */
                monthly_income_centavos: number;
                /** Format: int64 */
                growth_centavos: number;
                growth_bps: number;
            };
            /** @description Current-value share per asset class (fii, fixed_income, stocks, etfs). */
            allocation: {
                /** @enum {string} */
                asset_class: "fii" | "fixed_income" | "stocks" | "etfs";
                /** Format: int64 */
                value_centavos: number;
                share_bps: number;
            }[];
            /** @description Current-value share per FII sector, as a fraction of the FII total. */
            fii_sectors: {
                sector: string;
                /** Format: int64 */
                value_centavos: number;
                share_bps: number;
            }[];
            /** @description Held FIIs with no current quote, valued at cost basis. */
            stale_tickers: string[];
        };
        FixedIncomeRequest: {
            name: string;
            institution: string;
            /**
             * Format: int64
             * @description Invested amount in centavos (integer, never a float).
             */
            invested_amount_centavos: number;
            /**
             * @description Meaning depends on indexer_type (SPEC-109): the flat annual rate for
             *     prefixado; the percentage of CDI in bps-of-percent for cdi_percentual
             *     (12000 = 120%); the spread over IPCA in bps for ipca_spread (580 = +5.80%).
             */
            annual_rate_bps: number;
            /**
             * @description "" / omitted defaults to prefixado (SPEC-109 BR-1093).
             * @default prefixado
             * @enum {string}
             */
            indexer_type: "prefixado" | "cdi_percentual" | "ipca_spread";
            /**
             * Format: date
             * @description Maturity as `YYYY-MM-DD`; null/omitted for daily-liquidity holdings.
             */
            maturity_date?: string | null;
            /** @enum {string} */
            liquidity_type: "daily" | "at_maturity";
        };
        FixedIncomeResponse: {
            /** Format: uuid */
            id: string;
            name: string;
            institution: string;
            /** Format: int64 */
            invested_amount_centavos: number;
            /** @description The raw stored value — see FixedIncomeRequest.annual_rate_bps. */
            annual_rate_bps: number;
            /** @enum {string} */
            indexer_type?: "prefixado" | "cdi_percentual" | "ipca_spread";
            /**
             * @description Computed, never persisted (SPEC-109): the resolved current annual rate —
             *     equal to annual_rate_bps for prefixado; resolved against the latest
             *     SELIC/CDI/IPCA (GET /market/indicators) for cdi_percentual/ipca_spread.
             */
            effective_annual_rate_bps?: number;
            /** Format: date */
            maturity_date?: string | null;
            /** @enum {string} */
            liquidity_type: "daily" | "at_maturity";
            /** Format: date-time */
            created_at: string;
            /** Format: date-time */
            updated_at: string;
        };
        /** @description One latest observation of a SPEC-006 macro indicator (SPEC-109 FR-1095). */
        MarketIndicatorResponse: {
            /** @enum {string} */
            indicator: "selic" | "cdi" | "ipca";
            /**
             * Format: int64
             * @description The rate in basis points (1 bp = 0.01%) — never a float.
             */
            value_bps: number;
            /** Format: date */
            reference_date: string;
        };
    };
    responses: {
        /** @description The request body or a field is invalid. */
        BadRequest: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "error": "invalid request body"
                 *     }
                 */
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description Authentication is required or the session is invalid. */
        Unauthorized: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "error": "authentication required"
                 *     }
                 */
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description No such holding owned by the caller. */
        HoldingNotFound: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "error": "holding not found"
                 *     }
                 */
                "application/json": components["schemas"]["Error"];
            };
        };
        /** @description No such resource owned by the caller. */
        NotFound: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                /**
                 * @example {
                 *       "error": "thread not found"
                 *     }
                 */
                "application/json": components["schemas"]["Error"];
            };
        };
    };
    parameters: {
        /** @description The holding's UUID. */
        HoldingID: string;
    };
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    getHealthz: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The process is alive. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "status": "ok"
                     *     }
                     */
                    "application/json": components["schemas"]["StatusResponse"];
                };
            };
        };
    };
    getReadyz: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The app and its dependencies are ready. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "status": "ready",
                     *       "checks": {
                     *         "db": "up"
                     *       }
                     *     }
                     */
                    "application/json": components["schemas"]["ReadinessResponse"];
                };
            };
            /** @description A dependency is unavailable. */
            503: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "status": "not_ready",
                     *       "checks": {
                     *         "db": "down"
                     *       }
                     *     }
                     */
                    "application/json": components["schemas"]["ReadinessResponse"];
                };
            };
        };
    };
    getVersion: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Build version, commit, and build time. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["VersionResponse"];
                };
            };
        };
    };
    registerUser: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["Credentials"];
            };
        };
        responses: {
            /** @description Account created. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            400: components["responses"]["BadRequest"];
            /** @description The email is already registered. */
            409: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "error": "email already registered"
                     *     }
                     */
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    loginUser: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["Credentials"];
            };
        };
        responses: {
            /** @description Authenticated. A session cookie is set. */
            200: {
                headers: {
                    /** @description The HttpOnly session cookie (name defaults to `yf_session`). */
                    "Set-Cookie"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            400: components["responses"]["BadRequest"];
            /** @description Invalid email or password. */
            401: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "error": "invalid email or password"
                     *     }
                     */
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    logoutUser: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Session revoked; the cookie is cleared. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getCurrentUser: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The current user. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["User"];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getProfile: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The investor profile. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ProfileResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
            /** @description The profile has not been set. */
            404: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    /**
                     * @example {
                     *       "error": "profile not set"
                     *     }
                     */
                    "application/json": components["schemas"]["Error"];
                };
            };
        };
    };
    setProfile: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ProfileRequest"];
            };
        };
        responses: {
            /** @description The saved profile. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ProfileResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    listFIIHoldings: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The caller's FII holdings (possibly empty). */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["FIIHoldingResponse"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    createFIIHolding: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["FIIHoldingRequest"];
            };
        };
        responses: {
            /** @description The created holding. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["FIIHoldingResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    updateFIIHolding: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description The holding's UUID. */
                id: components["parameters"]["HoldingID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["FIIHoldingRequest"];
            };
        };
        responses: {
            /** @description The updated holding. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["FIIHoldingResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["HoldingNotFound"];
        };
    };
    deleteFIIHolding: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description The holding's UUID. */
                id: components["parameters"]["HoldingID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["HoldingNotFound"];
        };
    };
    listFixedIncomeHoldings: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The caller's fixed-income holdings (possibly empty). */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["FixedIncomeResponse"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    createFixedIncomeHolding: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["FixedIncomeRequest"];
            };
        };
        responses: {
            /** @description The created holding. */
            201: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["FixedIncomeResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    updateFixedIncomeHolding: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description The holding's UUID. */
                id: components["parameters"]["HoldingID"];
            };
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["FixedIncomeRequest"];
            };
        };
        responses: {
            /** @description The updated holding. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["FixedIncomeResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["HoldingNotFound"];
        };
    };
    deleteFixedIncomeHolding: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                /** @description The holding's UUID. */
                id: components["parameters"]["HoldingID"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Deleted. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["HoldingNotFound"];
        };
    };
    getDashboard: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The computed dashboard. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["DashboardResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getInsights: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The generated insights (possibly empty / degraded — see `available`). */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["InsightsResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    postRebalancing: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": {
                    /**
                     * Format: int64
                     * @description The amount to allocate, in centavos (> 0).
                     */
                    contribution_centavos: number;
                    /** @description Opt into the illustrative per-candidate within-area share (default false). */
                    include_asset_shares?: boolean;
                };
            };
        };
        responses: {
            /** @description The generated guidance (possibly degraded — see `available`). */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["RebalancingResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    getHealthScore: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The computed score + breakdown (+ narrative when available). */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["HealthScoreResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getProjections: {
        parameters: {
            query?: {
                /** @description New money contributed each month, in centavos (≥ 0). */
                monthly_contribution_centavos?: number;
                /** @description Projection horizon in years (1–40). */
                horizon_years?: number;
            };
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The computed income + net-worth projections. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ProjectionsResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
        };
    };
    postChatMessage: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": {
                    /** @description Optional — omit to start a new thread. */
                    thread_id?: string;
                    /** @description The user's message (1–2000 characters). */
                    content: string;
                };
            };
        };
        responses: {
            /** @description The gated assistant reply (possibly degraded — see `available`). */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ChatReplyResponse"];
                };
            };
            400: components["responses"]["BadRequest"];
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    listChatThreads: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The caller's threads, most-recently-updated first. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ChatThreadResponse"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    clearChatThreads: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description All threads cleared. */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    getChatThread: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The thread and its ordered messages. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["ChatThreadDetailResponse"];
                };
            };
            401: components["responses"]["Unauthorized"];
            404: components["responses"]["NotFound"];
        };
    };
    deleteChatThread: {
        parameters: {
            query?: never;
            header?: never;
            path: {
                id: string;
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The thread was deleted (or did not exist — no existence oracle). */
            204: {
                headers: {
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Unauthorized"];
        };
    };
    listMarketIndicators: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The latest observation per indicator (possibly a subset, or empty). */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["MarketIndicatorResponse"][];
                };
            };
            401: components["responses"]["Unauthorized"];
        };
    };
}
