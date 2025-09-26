import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";

/**
 * A hook for querying your custom app data.
 * @desc A thin wrapper around standard fetch and react-query's useQuery.
 *
 * @param {Object} options - The options for your query. Accepts 3 keys:
 *
 * 1. url: The URL to query. E.g: /api/widgets/1`
 * 2. fetchInit: The init options for fetch.  See: https://developer.mozilla.org/en-US/docs/Web/API/fetch#parameters
 * 3. reactQueryOptions: The options for `useQuery`. See: https://react-query.tanstack.com/reference/useQuery
 *
 * @returns Return value of useQuery.  See: https://react-query.tanstack.com/reference/useQuery.
 */
export const useAppQuery = ({ url, fetchInit = {}, reactQueryOptions }) => {

  const fetchFn = useMemo(() => {
    return async () => {
      // Add Shopify authentication headers
      const searchParams = new URLSearchParams(window.location.search);
      const idToken = searchParams.get('id_token');
      const session = searchParams.get('session');

      const authHeaders = {};
      if (idToken) {
        authHeaders['Authorization'] = `Bearer ${idToken}`;
      } else if (session) {
        authHeaders['Authorization'] = `Bearer ${session}`;
      }

      const mergedFetchInit = {
        ...fetchInit,
        headers: {
          ...authHeaders,
          ...fetchInit.headers,
        },
      };
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 10000); // 10 second timeout

      const response = await fetch(url, {
        ...mergedFetchInit,
        signal: controller.signal
      });
      clearTimeout(timeoutId);

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP error! status: ${response.status} - ${errorText}`);
      }
      const data = await response.json();
      return data;
    };
  }, [url, JSON.stringify(fetchInit)]);

  return useQuery({
    queryKey: [url],
    queryFn: fetchFn,
    ...reactQueryOptions,
  });
};
