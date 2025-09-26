import { useState } from "react";
import {
  Card,
  BlockStack,
  Text,
} from "@shopify/polaris";
import { useAppBridge } from "@shopify/app-bridge-react";
import { useAppQuery } from "../hooks";

export function ProductsCard() {
  const app = useAppBridge();
  const [isLoading, setIsLoading] = useState(false);

  const {
    data,
    refetch: refetchProductCount,
    isLoading: isLoadingCount,
  } = useAppQuery({
    url: "/api/products/count",
    reactQueryOptions: {
      retry: 1,
      refetchOnWindowFocus: false,
      gcTime: 5 * 60 * 1000, // 5 minutes
      staleTime: 30 * 1000, // 30 seconds
    },
  });


  const handlePopulate = async () => {
    setIsLoading(true);
    try {
      // Get session token from URL parameters (same as useAppQuery)
      const searchParams = new URLSearchParams(window.location.search);
      const idToken = searchParams.get('id_token');
      const session = searchParams.get('session');

      const authHeaders = {};
      if (idToken) {
        authHeaders['Authorization'] = `Bearer ${idToken}`;
      } else if (session) {
        authHeaders['Authorization'] = `Bearer ${session}`;
      }

      const response = await fetch("/api/products/create", {
        method: "GET",
        headers: {
          ...authHeaders,
          "Content-Type": "application/json"
        }
      });

      if (response.ok) {
        await refetchProductCount();
        if (app) {
          app.toast.show("5 products created!");
        }
      } else {
        if (app) {
          app.toast.show("There was an error creating products", { isError: true });
        }
      }
    } catch (error) {
      if (app) {
        app.toast.show("There was an error creating products", { isError: true });
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Card>
      <BlockStack gap="300">
        <Text variant="headingMd" as="h2">
          Product Counter
        </Text>
        <Text as="p">
          Sample products are created with a default title and price. You can
          remove them at any time.
        </Text>
        <div style={{ textAlign: 'center' }}>
          <BlockStack gap="300">
            <BlockStack gap="200">
              <Text variant="headingMd" as="h4">
                TOTAL PRODUCTS
              </Text>
              <Text variant="heading2xl" as="p">
                {isLoadingCount || !data ? "-" : (data?.count ?? 0)}
              </Text>
            </BlockStack>
            <button
              onClick={handlePopulate}
              disabled={isLoading || isLoadingCount}
              style={{
                padding: '12px 20px',
                backgroundColor: '#008060',
                color: 'white',
                border: 'none',
                borderRadius: '6px',
                cursor: isLoading || isLoadingCount ? 'not-allowed' : 'pointer',
                opacity: isLoading || isLoadingCount ? 0.6 : 1,
              }}
            >
              {isLoading ? 'Creating...' : 'Populate 5 products'}
            </button>
          </BlockStack>
        </div>
      </BlockStack>
    </Card>
  );
}
