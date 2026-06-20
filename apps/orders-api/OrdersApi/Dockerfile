

# ----- Build stage --------------------------------------------------
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src

# Copy project file first for layer caching
COPY OrdersApi.csproj .
RUN dotnet restore

# Copy the rest and publish
COPY . .
RUN dotnet publish -c Release -o /app --no-restore

# ----- Runtime stage ------------------------------------------------
# chiseled image: minimal, non-root by default, no shell, no package manager
FROM mcr.microsoft.com/dotnet/aspnet:8.0-jammy-chiseled AS runtime
WORKDIR /app
COPY --from=build /app .

EXPOSE 8080
ENV ASPNETCORE_URLS=http://+:8080
USER $APP_UID

ENTRYPOINT ["dotnet", "OrdersApi.dll"]